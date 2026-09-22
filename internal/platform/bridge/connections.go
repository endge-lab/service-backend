// Package bridge implements bounded WebSocket transport; it owns no debug authorization.
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

const MaxMessageBytes = 16 * 1024 * 1024
const pingInterval = 15 * time.Second
const pongTimeout = 60 * time.Second
const writeTimeout = 5 * time.Second

// Socket is the transport boundary implemented by the HTTP WebSocket adapter.
type Socket interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	WriteControl(int, []byte, time.Time) error
	SetReadLimit(int64)
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	SetPongHandler(func(string) error)
	Close() error
}

type connection struct {
	socket      Socket
	queue       chan []byte
	done        chan struct{}
	once        sync.Once
	mu          sync.Mutex
	queuedBytes int
}

func (c *connection) close() { c.once.Do(func() { close(c.done); _ = c.socket.Close() }) }

// Connections owns live sockets, bounded writes, heartbeat and deterministic cleanup.
type Connections struct {
	mu      sync.Mutex
	peers   map[string]*connection
	stopped bool
	wg      sync.WaitGroup
}

func NewConnections() *Connections { return &Connections{peers: make(map[string]*connection)} }

// Serve registers one socket and runs a single reader and writer until either fails.
func (r *Connections) Serve(id string, socket Socket, onMessage func([]byte) error, onCheck func(context.Context) bool, onClose func()) error {
	c := &connection{socket: socket, queue: make(chan []byte, 32), done: make(chan struct{})}
	r.mu.Lock()
	if r.stopped || len(r.peers) >= 256 {
		r.mu.Unlock()
		c.close()
		return fmt.Errorf("Bridge connection limit reached")
	}
	r.peers[id] = c
	r.wg.Add(1)
	r.mu.Unlock()
	defer r.wg.Done()
	writerDone := make(chan struct{})
	defer func() {
		c.close()
		<-writerDone
		r.mu.Lock()
		delete(r.peers, id)
		r.mu.Unlock()
		onClose()
	}()
	socket.SetReadLimit(MaxMessageBytes)
	_ = socket.SetReadDeadline(time.Now().Add(10 * time.Second))
	registered := false
	socket.SetPongHandler(func(string) error {
		if !registered {
			return nil
		}
		return socket.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	go func() {
		defer close(writerDone)
		defer c.close()
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-c.done:
				return
			case payload := <-c.queue:
				c.mu.Lock()
				c.queuedBytes -= len(payload)
				c.mu.Unlock()
				if err := socket.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
					return
				}
				if err := socket.WriteMessage(1, payload); err != nil {
					return
				}
			case <-ticker.C:
				if err := socket.WriteControl(9, nil, time.Now().Add(writeTimeout)); err != nil {
					return
				}
				// Browser JS cannot observe protocol Ping/Pong; this bounds its local stale session.
				if !r.Send(id, entities.BridgeMessage{Type: "heartbeat"}) {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				valid := onCheck(ctx)
				cancel()
				if !valid {
					return
				}
			}
		}
	}()
	for {
		kind, payload, err := socket.ReadMessage()
		if err != nil {
			return err
		}
		if kind != 1 {
			return fmt.Errorf("Only text messages are supported")
		}
		if err := onMessage(payload); err != nil {
			return err
		}
		if !registered {
			registered = true
			_ = socket.SetReadDeadline(time.Now().Add(pongTimeout))
		}
	}
}

// Send enqueues without blocking. Slow receivers are closed instead of retaining payloads.
func (r *Connections) Send(id string, message entities.BridgeMessage) bool {
	r.mu.Lock()
	c := r.peers[id]
	r.mu.Unlock()
	if c == nil {
		return false
	}
	payload, err := json.Marshal(message)
	if err != nil || len(payload) > MaxMessageBytes {
		c.close()
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.done:
		return false
	default:
	}
	if c.queuedBytes+len(payload) > 2*MaxMessageBytes {
		c.close()
		return false
	}
	select {
	case c.queue <- payload:
		c.queuedBytes += len(payload)
		return true
	default:
		c.close()
		return false
	}
}

func (r *Connections) Close(id string) {
	r.mu.Lock()
	c := r.peers[id]
	r.mu.Unlock()
	if c != nil {
		c.close()
	}
}

// Shutdown rejects new sockets and waits for all connection cleanup within host deadline.
func (r *Connections) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	r.stopped = true
	for _, c := range r.peers {
		c.close()
	}
	r.mu.Unlock()
	done := make(chan struct{})
	go func() { r.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
