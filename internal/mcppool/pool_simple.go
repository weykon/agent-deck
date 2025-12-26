package mcppool

import (
	"context"
	
	"log"
	"sync"
)

type Pool struct {
	proxies map[string]*SocketProxy
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	config  *PoolConfig
}

type PoolConfig struct {
	Enabled        bool
	PoolAll        bool
	ExcludeMCPs    []string
	PoolMCPs       []string
	FallbackStdio  bool
}

func NewPool(ctx context.Context, config *PoolConfig) (*Pool, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Pool{
		proxies: make(map[string]*SocketProxy),
		ctx:     ctx,
		cancel:  cancel,
		config:  config,
	}, nil
}

func (p *Pool) Start(name, command string, args []string, env map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.proxies[name]; exists {
		return nil
	}

	proxy, err := NewSocketProxy(p.ctx, name, command, args, env)
	if err != nil {
		return err
	}

	if err := proxy.Start(); err != nil {
		return err
	}

	p.proxies[name] = proxy
	return nil
}

func (p *Pool) ShouldPool(mcpName string) bool {
	if !p.config.Enabled {
		return false
	}

	if p.config.PoolAll {
		for _, excluded := range p.config.ExcludeMCPs {
			if excluded == mcpName {
				return false
			}
		}
		return true
	}

	for _, name := range p.config.PoolMCPs {
		if name == mcpName {
			return true
		}
	}
	return false
}

func (p *Pool) IsRunning(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	proxy, exists := p.proxies[name]
	return exists && proxy.Status == StatusRunning
}

func (p *Pool) GetURL(name string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if proxy, exists := p.proxies[name]; exists {
		return proxy.GetSocketPath()
	}
	return ""
}

func (p *Pool) GetSocketPath(name string) string {
	return p.GetURL(name)
}

func (p *Pool) Shutdown() error {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for name, proxy := range p.proxies {
		log.Printf("Stopping socket proxy: %s", name)
		_ = proxy.Stop()
	}

	return nil
}

func (p *Pool) ListServers() []ProxyInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	list := []ProxyInfo{}
	for _, proxy := range p.proxies {
		list = append(list, ProxyInfo{
			Name:        proxy.name,
			SocketPath:  proxy.socketPath,
			Status:      proxy.Status.String(),
			Clients:     proxy.GetClientCount(),
		})
	}
	return list
}

type ProxyInfo struct {
	Name       string
	SocketPath string
	Status     string
	Clients    int
}
