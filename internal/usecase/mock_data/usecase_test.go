package mock_data

import (
	"context"
	"encoding/json"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	errs "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"sync"
	"testing"
	"time"
)

type fakeGateway struct {
	mu           sync.Mutex
	next         int
	stopped      chan string
	create       func(context.Context) (entities.MockStream, error)
	stop         func(context.Context) error
	subscribe    func(context.Context) (ports.MockSubscription, error)
	keepaliveErr error
	idle         time.Duration
}

func (g *fakeGateway) ServiceInfo(context.Context) (ports.ConnectedServiceInfo, error) {
	return ports.ConnectedServiceInfo{Version: "0.1.0"}, nil
}
func (g *fakeGateway) Capabilities(context.Context) (json.RawMessage, error) {
	return json.RawMessage(`{"available":true,"idleTimeoutMs":1000,"readyTimeoutMs":1000,"limits":{"sessions":20,"sessionsPerOwner":2,"bufferBytes":1024,"requestBytes":512}}`), nil
}
func (g *fakeGateway) Generate(context.Context, entities.MockOwner, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{"items":[1]}`), nil
}
func (g *fakeGateway) Create(ctx context.Context, _ entities.MockOwner, _ json.RawMessage) (entities.MockStream, error) {
	if g.create != nil {
		return g.create(ctx)
	}
	return entities.MockStream{ID: "upstream", IdleTimeoutMs: g.idle.Milliseconds(), ExpiresAt: time.Now().Add(time.Second)}, nil
}
func (g *fakeGateway) Get(context.Context, entities.MockOwner, string) (entities.MockStream, error) {
	return entities.MockStream{ID: "upstream", IdleTimeoutMs: g.idle.Milliseconds()}, nil
}
func (g *fakeGateway) Update(ctx context.Context, o entities.MockOwner, id string, _ json.RawMessage) (entities.MockStream, error) {
	return g.Get(ctx, o, id)
}
func (g *fakeGateway) KeepAlive(ctx context.Context, o entities.MockOwner, id string) (entities.MockStream, error) {
	if g.keepaliveErr != nil {
		return entities.MockStream{}, g.keepaliveErr
	}
	return g.Get(ctx, o, id)
}
func (g *fakeGateway) Stop(ctx context.Context, _ entities.MockOwner, id string) error {
	if g.stop != nil {
		return g.stop(ctx)
	}
	select {
	case g.stopped <- id:
	default:
	}
	return nil
}
func (g *fakeGateway) Subscribe(ctx context.Context, _ entities.MockOwner, _ string) (ports.MockSubscription, error) {
	if g.subscribe != nil {
		return g.subscribe(ctx)
	}
	return &fakeSub{ctx: ctx}, nil
}

func TestSubscribeLeaseStartsBeforeAcknowledgement(t *testing.T) {
	u, g := setup(t)
	ctx := actor("user", "workspace", "viewer")
	info, e := u.Create(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	var acknowledged time.Time
	g.subscribe = func(ctx context.Context) (ports.MockSubscription, error) {
		time.Sleep(20 * time.Millisecond)
		acknowledged = time.Now()
		return &fakeSub{ctx: ctx}, nil
	}
	sub, e := u.Subscribe(ctx, info.ID)
	if e != nil {
		t.Fatal(e)
	}
	defer sub.Close()
	s, e := u.find(ctx, info.ID)
	if e != nil {
		t.Fatal(e)
	}
	s.mu.Lock()
	expires := s.expires
	s.mu.Unlock()
	if !expires.Before(acknowledged.Add(g.idle)) {
		t.Fatal("backend lease exceeds upstream lease")
	}
}

type fakeSub struct{ ctx context.Context }

func (s *fakeSub) Recv() (*entities.MockEvent, error) { <-s.ctx.Done(); return nil, s.ctx.Err() }
func (s *fakeSub) Close()                             {}
func actor(id, workspace, role string) context.Context {
	ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: id}})
	return entities.WithWorkspaceAccess(ctx, entities.WorkspaceAccess{Workspace: entities.Workspace{ID: workspace}, Role: role})
}
func setup(t *testing.T) (*UseCase, *fakeGateway) {
	g := &fakeGateway{stopped: make(chan string, 100), idle: 180 * time.Second}
	u := NewUseCase(g, DefaultConfig())
	t.Cleanup(u.Close)
	return u, g
}
func TestOwnershipQuotaAndCapabilities(t *testing.T) {
	u, _ := setup(t)
	ctx := actor("user", "workspace", "viewer")
	info, e := u.Create(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if info.ID == "upstream" || info.EventsURL == "" {
		t.Fatal("internal id exposed")
	}
	for _, other := range []context.Context{actor("foreign", "workspace", "viewer"), actor("user", "foreign", "viewer")} {
		if _, e = u.Get(other, info.ID); errs.HTTPStatusOf(e) != 404 {
			t.Fatal("foreign session visible", e)
		}
	}
	if _, e = u.Create(actor("user", "workspace", "guest"), nil); errs.HTTPStatusOf(e) != 403 {
		t.Fatal("non-viewer", e)
	}
	for range 4 {
		if _, e = u.Create(ctx, nil); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = u.Create(ctx, nil); errs.HTTPStatusOf(e) != 429 {
		t.Fatal("quota", e)
	}
	raw, e := u.Capabilities(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var caps map[string]any
	json.Unmarshal(raw, &caps)
	if caps["idleTimeoutMs"] != float64(1000) {
		t.Fatal("upstream lease not respected", string(raw))
	}
}
func TestPartialCreateCancellationCleansUp(t *testing.T) {
	u, g := setup(t)
	ctx, cancel := context.WithCancel(actor("user", "workspace", "viewer"))
	g.create = func(context.Context) (entities.MockStream, error) {
		cancel()
		return entities.MockStream{ID: "orphan", IdleTimeoutMs: 180000, ExpiresAt: time.Now().Add(time.Second)}, nil
	}
	if _, e := u.Create(ctx, nil); e == nil {
		t.Fatal("canceled create accepted")
	}
	select {
	case id := <-g.stopped:
		if id != "orphan" {
			t.Fatal(id)
		}
	case <-time.After(time.Second):
		t.Fatal("orphan not stopped")
	}
	if n, _ := u.Stats(); n != 0 {
		t.Fatal("public session leaked")
	}
}
func TestLeaseDoesNotRenewOnFailedKeepAliveAndCannotRevive(t *testing.T) {
	u, g := setup(t)
	ctx := actor("user", "workspace", "viewer")
	info, _ := u.Create(ctx, nil)
	sub, e := u.Subscribe(ctx, info.ID)
	if e != nil {
		t.Fatal(e)
	}
	defer sub.Close()
	s, _ := u.find(ctx, info.ID)
	s.mu.Lock()
	before := s.expires
	s.mu.Unlock()
	g.keepaliveErr = errs.New("mock.unavailable", "offline", 503)
	if _, e = u.KeepAlive(ctx, info.ID); e == nil {
		t.Fatal("offline keepalive accepted")
	}
	s.mu.Lock()
	if s.expires != before {
		t.Fatal("failed request renewed lease")
	}
	s.expires = time.Now().Add(-time.Second)
	s.mu.Unlock()
	g.keepaliveErr = nil
	if _, e = u.KeepAlive(ctx, info.ID); errs.HTTPStatusOf(e) != 404 {
		t.Fatal("revived expired lease", e)
	}
	if n, _ := u.Stats(); n != 0 {
		t.Fatal("expired session retained")
	}
}
func TestConcurrentSubscribeAndStop(t *testing.T) {
	u, _ := setup(t)
	ctx := actor("user", "workspace", "viewer")
	info, _ := u.Create(ctx, nil)
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub, e := u.Subscribe(ctx, info.ID)
			if e == nil {
				sub.Close()
			}
		}()
	}
	wg.Wait()
	u.Stop(ctx, info.ID)
	if n, _ := u.Stats(); n != 0 {
		t.Fatal("session leaked")
	}
}

func TestSlowCleanupRetainsAdmission(t *testing.T) {
	u, g := setup(t)
	u.config.Sessions = 2
	u.config.PerOwner = 2
	release := make(chan struct{})
	defer close(release)
	g.stop = func(ctx context.Context) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	ctx := actor("user", "workspace", "viewer")
	for range 2 {
		s, e := u.Create(ctx, nil)
		if e != nil {
			t.Fatal(e)
		}
		if e = u.Stop(ctx, s.ID); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := u.Create(ctx, nil); errs.HTTPStatusOf(e) != 429 {
		t.Fatal("cleanup must remain bounded by admission", e)
	}
}
