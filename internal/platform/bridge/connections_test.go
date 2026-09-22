package bridge

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/domain/entities"
	clientws "github.com/fasthttp/websocket"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func TestRealSocketDisconnectReleasesRegistryAndWriter(t *testing.T) {
	registry := NewConnections()
	closed := make(chan struct{})
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/bridge", websocket.New(func(socket *websocket.Conn) {
		_ = registry.Serve("connection", socket, func([]byte) error {
			registry.Send("connection", entities.BridgeMessage{Type: "ack"})
			return nil
		}, func(context.Context) bool { return true }, func() { close(closed) })
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan struct{})
	go func() { defer close(serveDone); _ = app.Listener(listener) }()
	defer func() { _ = app.ShutdownWithTimeout(time.Second); <-serveDone }()
	socket, _, err := clientws.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/bridge", http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	_ = socket.SetReadDeadline(time.Now().Add(time.Second))
	if err = socket.WriteMessage(clientws.TextMessage, []byte(`{"type":"hello"}`)); err != nil {
		t.Fatal(err)
	}
	_, payload, err := socket.ReadMessage()
	if err != nil || string(payload) != `{"type":"ack"}` {
		t.Fatalf("unexpected response %s: %v", payload, err)
	}
	_ = socket.Close()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup did not finish")
	}
	if registry.Send("connection", entities.BridgeMessage{Type: "late"}) {
		t.Fatal("closed socket retained")
	}
	registry.mu.Lock()
	remaining := len(registry.peers)
	registry.mu.Unlock()
	if remaining != 0 {
		t.Fatal("registry retained closed connection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := registry.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}
