// Quick proof-of-concept: Unix socket proxy for stdio MCP
// Usage: go run test-socket-proxy.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
)

func main() {
	socketPath := "/tmp/test-mcp-memory.sock"

	// Clean up old socket
	os.Remove(socketPath)

	// Start MCP process (stdio)
	cmd := exec.Command("npx", "-y", "@modelcontextprotocol/server-memory")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		log.Fatal("Failed to start MCP:", err)
	}

	log.Printf("Started MCP process (PID: %d)", cmd.Process.Pid)

	// Forward stderr to our stderr for debugging
	go io.Copy(os.Stderr, stderr)

	// Create Unix socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("Failed to create socket:", err)
	}
	defer listener.Close()

	log.Printf("Socket listening at: %s", socketPath)
	log.Printf("Test with: nc -U %s", socketPath)

	var clientsMu sync.Mutex
	clients := make(map[string]net.Conn)
	clientCounter := 0

	// Accept connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			clientID := fmt.Sprintf("client-%d", clientCounter)
			clientCounter++

			clientsMu.Lock()
			clients[clientID] = conn
			clientsMu.Unlock()

			log.Printf("Client connected: %s", clientID)

			// Read from client, write to MCP stdin
			go func(id string, c net.Conn) {
				scanner := bufio.NewScanner(c)
				for scanner.Scan() {
					line := scanner.Text()
					log.Printf("[%s → MCP] %s", id, line)
					fmt.Fprintln(stdin, line)
				}

				clientsMu.Lock()
				delete(clients, id)
				clientsMu.Unlock()

				log.Printf("Client disconnected: %s", id)
				c.Close()
			}(clientID, conn)
		}
	}()

	// Broadcast MCP stdout to all clients
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("[MCP → ALL] %s", line)

			clientsMu.Lock()
			for id, conn := range clients {
				log.Printf("  → Sending to %s", id)
				fmt.Fprintln(conn, line)
			}
			clientsMu.Unlock()
		}
	}()

	// Wait for process to exit
	cmd.Wait()
	log.Printf("MCP process exited")
}
