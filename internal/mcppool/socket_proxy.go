package mcppool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

// SocketProxy wraps a stdio MCP process with a Unix socket
type SocketProxy struct {
	name       string
	socketPath string
	command    string
	args       []string
	env        map[string]string

	mcpProcess *exec.Cmd
	mcpStdin   io.WriteCloser
	mcpStdout  io.ReadCloser

	listener net.Listener

	clients   map[string]net.Conn
	clientsMu sync.RWMutex

	requestMap map[interface{}]string
	requestMu  sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc

	logFile   string
	logWriter io.WriteCloser

	Status ServerStatus
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

func NewSocketProxy(ctx context.Context, name, command string, args []string, env map[string]string) (*SocketProxy, error) {
	ctx, cancel := context.WithCancel(ctx)
	socketPath := filepath.Join("/tmp", fmt.Sprintf("agentdeck-mcp-%s.sock", name))
	os.Remove(socketPath)

	return &SocketProxy{
		name:       name,
		socketPath: socketPath,
		command:    command,
		args:       args,
		env:        env,
		clients:    make(map[string]net.Conn),
		requestMap: make(map[interface{}]string),
		ctx:        ctx,
		cancel:     cancel,
		Status:     StatusStarting,
	}, nil
}

func (p *SocketProxy) Start() error {
	logDir := filepath.Join(os.Getenv("HOME"), ".agent-deck", "logs", "mcppool")
	_ = os.MkdirAll(logDir, 0755)
	p.logFile = filepath.Join(logDir, fmt.Sprintf("%s_socket.log", p.name))

	logWriter, err := os.Create(p.logFile)
	if err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	p.logWriter = logWriter

	p.mcpProcess = exec.CommandContext(p.ctx, p.command, p.args...)
	cmdEnv := os.Environ()
	for k, v := range p.env {
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
	}
	p.mcpProcess.Env = cmdEnv

	p.mcpStdin, err = p.mcpProcess.StdinPipe()
	if err != nil {
		return err
	}
	p.mcpStdout, err = p.mcpProcess.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, _ := p.mcpProcess.StderrPipe()

	if err := p.mcpProcess.Start(); err != nil {
		return err
	}

	log.Printf("Started MCP %s (PID: %d)", p.name, p.mcpProcess.Process.Pid)
	go func() { _, _ = io.Copy(p.logWriter, stderr) }()

	listener, err := net.Listen("unix", p.socketPath)
	if err != nil {
		_ = p.mcpProcess.Process.Kill()
		return err
	}
	p.listener = listener

	log.Printf("Socket proxy %s at: %s", p.name, p.socketPath)

	go p.acceptConnections()
	go p.broadcastResponses()

	p.Status = StatusRunning
	return nil
}

func (p *SocketProxy) acceptConnections() {
	clientCounter := 0
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.ctx.Done():
				return
			default:
				continue
			}
		}

		sessionID := fmt.Sprintf("%s-client-%d", p.name, clientCounter)
		clientCounter++

		p.clientsMu.Lock()
		p.clients[sessionID] = conn
		p.clientsMu.Unlock()

		log.Printf("[%s] Client connected: %s", p.name, sessionID)
		go p.handleClient(sessionID, conn)
	}
}

func (p *SocketProxy) handleClient(sessionID string, conn net.Conn) {
	defer func() {
		p.clientsMu.Lock()
		delete(p.clients, sessionID)
		p.clientsMu.Unlock()
		conn.Close()
		log.Printf("[%s] Client disconnected: %s", p.name, sessionID)
	}()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		if req.ID != nil {
			p.requestMu.Lock()
			p.requestMap[req.ID] = sessionID
			p.requestMu.Unlock()
		}

		_, _ = p.mcpStdin.Write(line)
		_, _ = p.mcpStdin.Write([]byte("\n"))
	}
}

func (p *SocketProxy) broadcastResponses() {
	scanner := bufio.NewScanner(p.mcpStdout)
	for scanner.Scan() {
		line := scanner.Bytes()

		var resp JSONRPCResponse
		if json.Unmarshal(line, &resp) != nil {
			p.broadcastToAll(line)
			continue
		}

		if resp.ID != nil {
			p.routeToClient(resp.ID, line)
		} else {
			p.broadcastToAll(line)
		}
	}
}

func (p *SocketProxy) routeToClient(responseID interface{}, line []byte) {
	p.requestMu.Lock()
	sessionID, exists := p.requestMap[responseID]
	if exists {
		delete(p.requestMap, responseID)
	}
	p.requestMu.Unlock()

	if !exists {
		p.broadcastToAll(line)
		return
	}

	p.clientsMu.RLock()
	conn, exists := p.clients[sessionID]
	p.clientsMu.RUnlock()

	if exists {
		_, _ = conn.Write(line)
		_, _ = conn.Write([]byte("\n"))
	}
}

func (p *SocketProxy) broadcastToAll(line []byte) {
	p.clientsMu.RLock()
	defer p.clientsMu.RUnlock()

	for _, conn := range p.clients {
		_, _ = conn.Write(line)
		_, _ = conn.Write([]byte("\n"))
	}
}

func (p *SocketProxy) Stop() error {
	p.cancel()
	if p.listener != nil {
		p.listener.Close()
	}
	if p.mcpProcess != nil {
		p.mcpStdin.Close()
		_ = p.mcpProcess.Process.Signal(syscall.SIGTERM)
		_ = p.mcpProcess.Wait()
	}
	os.Remove(p.socketPath)
	if p.logWriter != nil {
		p.logWriter.Close()
	}
	p.Status = StatusStopped
	return nil
}

func (p *SocketProxy) GetSocketPath() string {
	return p.socketPath
}

func (p *SocketProxy) GetClientCount() int {
	p.clientsMu.RLock()
	defer p.clientsMu.RUnlock()
	return len(p.clients)
}

func (p *SocketProxy) HealthCheck() error {
	if p.mcpProcess == nil {
		return fmt.Errorf("process not running")
	}
	if err := p.mcpProcess.Process.Signal(syscall.Signal(0)); err != nil {
		return err
	}
	if _, err := os.Stat(p.socketPath); err != nil {
		return err
	}
	return nil
}
