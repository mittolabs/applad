package runtime

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Container represents a running function container in the pool.
type Container struct {
	ID        string
	FuncID    string
	ImageName string
	Port      string // host port or IP:port to reach the container
	CreatedAt time.Time
	LastUsed  time.Time
	InUse     bool
}

// Pool manages warm function containers.
type Pool struct {
	mu         sync.RWMutex
	containers map[string]*Container // containerID -> Container
	byFunc     map[string][]string   // functionID -> []containerID (warm pool)
	docker     *Client
	maxIdle    time.Duration
}

// NewPool creates a container pool manager.
func NewPool(docker *Client) *Pool {
	p := &Pool{
		containers: make(map[string]*Container),
		byFunc:     make(map[string][]string),
		docker:     docker,
		maxIdle:    5 * time.Minute,
	}
	// Start background reaper
	go p.reapLoop()
	return p
}

// GetOrCreate returns a warm container for the function, or creates one.
func (p *Pool) GetOrCreate(ctx context.Context, funcID, imageName string, env []string) (*Container, error) {
	// Try to find a warm container
	p.mu.Lock()
	if ids, ok := p.byFunc[funcID]; ok {
		for _, id := range ids {
			c := p.containers[id]
			if c != nil && !c.InUse {
				c.InUse = true
				c.LastUsed = time.Now()
				p.mu.Unlock()
				return c, nil
			}
		}
	}
	p.mu.Unlock()

	// No warm container available — create one
	return p.create(ctx, funcID, imageName, env)
}

func (p *Pool) create(ctx context.Context, funcID, imageName string, env []string) (*Container, error) {
	name := fmt.Sprintf("applad-fn-%s-%d", funcID[:8], time.Now().UnixNano()%1e6)

	containerID, err := p.docker.CreateContainer(ctx, name, ContainerConfig{
		Image: imageName,
		Env:   env,
		Labels: map[string]string{
			"applad.function": funcID,
			"applad.managed":  "true",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("pool: create container: %w", err)
	}

	if err := p.docker.StartContainer(ctx, containerID); err != nil {
		p.docker.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("pool: start container: %w", err)
	}

	// Wait for container to be ready (HTTP server up)
	port, err := p.waitForReady(ctx, containerID)
	if err != nil {
		p.docker.StopContainer(ctx, containerID)
		p.docker.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("pool: container not ready: %w", err)
	}

	c := &Container{
		ID:        containerID,
		FuncID:    funcID,
		ImageName: imageName,
		Port:      port,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		InUse:     true,
	}

	p.mu.Lock()
	p.containers[containerID] = c
	p.byFunc[funcID] = append(p.byFunc[funcID], containerID)
	p.mu.Unlock()

	return c, nil
}

// Release marks a container as available for reuse.
func (p *Pool) Release(containerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.containers[containerID]; ok {
		c.InUse = false
		c.LastUsed = time.Now()
	}
}

// Destroy immediately removes a container from the pool.
func (p *Pool) Destroy(ctx context.Context, containerID string) {
	p.mu.Lock()
	c, ok := p.containers[containerID]
	if ok {
		delete(p.containers, containerID)
		// Remove from byFunc
		if ids, ok := p.byFunc[c.FuncID]; ok {
			for i, id := range ids {
				if id == containerID {
					p.byFunc[c.FuncID] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
		}
	}
	p.mu.Unlock()

	if ok {
		p.docker.StopContainer(ctx, containerID)
		p.docker.RemoveContainer(ctx, containerID)
	}
}

// DestroyFunction removes all containers for a function.
func (p *Pool) DestroyFunction(ctx context.Context, funcID string) {
	p.mu.Lock()
	ids := p.byFunc[funcID]
	delete(p.byFunc, funcID)
	for _, id := range ids {
		delete(p.containers, id)
	}
	p.mu.Unlock()

	for _, id := range ids {
		p.docker.StopContainer(ctx, id)
		p.docker.RemoveContainer(ctx, id)
	}
}

// waitForReady polls the container until its HTTP server responds.
func (p *Pool) waitForReady(ctx context.Context, containerID string) (string, error) {
	deadline := time.After(30 * time.Second)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline:
			return "", fmt.Errorf("timeout waiting for container to start")
		case <-tick.C:
			port, err := p.docker.GetContainerPort(ctx, containerID)
			if err != nil {
				continue
			}

			// Try to reach the HTTP server
			addr := "http://localhost:" + port
			if len(port) > 5 { // IP:port format
				addr = "http://" + port
			}

			client := &http.Client{Timeout: 500 * time.Millisecond}
			resp, err := client.Get(addr + "/health")
			if err != nil {
				// Try just connecting
				resp, err = client.Get(addr + "/")
				if err != nil {
					continue
				}
			}
			resp.Body.Close()
			return port, nil
		}
	}
}

// reapLoop periodically removes idle containers.
func (p *Pool) reapLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		p.reap()
	}
}

func (p *Pool) reap() {
	ctx := context.Background()
	now := time.Now()
	var toRemove []string

	p.mu.RLock()
	for id, c := range p.containers {
		if !c.InUse && now.Sub(c.LastUsed) > p.maxIdle {
			toRemove = append(toRemove, id)
		}
	}
	p.mu.RUnlock()

	for _, id := range toRemove {
		log.Printf("pool: reaping idle container %s", id[:12])
		p.Destroy(ctx, id)
	}
}
