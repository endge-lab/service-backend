package mock_data

import (
	"context"
	"encoding/json"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	errs "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"github.com/endge-lab/service-backend/internal/usecase/shared"
	"github.com/google/uuid"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	Sessions, PerOwner                      int
	ReadyTimeout, IdleTimeout, WriteTimeout time.Duration
	BufferBytes                             int64
	RequestBytes                            int
}

func DefaultConfig() Config {
	return Config{Sessions: 32, PerOwner: 5, ReadyTimeout: 30 * time.Second, IdleTimeout: 180 * time.Second, WriteTimeout: 10 * time.Second, BufferBytes: 64 << 20, RequestBytes: 2 << 20}
}

type session struct {
	mu               sync.Mutex
	owner            entities.MockOwner
	public, upstream string
	expires          time.Time
	subscribed       bool
	idle             time.Duration
	ctx              context.Context
	cancel           context.CancelFunc
	once             sync.Once
}
type UseCase struct {
	gateway  ports.MockGeneratorGateway
	config   Config
	mu       sync.Mutex
	sessions map[string]*session
	pending  map[entities.MockOwner]int
	ctx      context.Context
	cancel   context.CancelFunc
	buffered atomic.Int64
}

func NewUseCase(g ports.MockGeneratorGateway, c Config) *UseCase {
	ctx, cancel := context.WithCancel(context.Background())
	return &UseCase{gateway: g, config: c, sessions: map[string]*session{}, pending: map[entities.MockOwner]int{}, ctx: ctx, cancel: cancel}
}
func (u *UseCase) Start() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-u.ctx.Done():
				return
			case <-ticker.C:
				u.reap()
			}
		}
	}()
}
func (u *UseCase) Close() {
	u.cancel()
	u.mu.Lock()
	for _, s := range u.sessions {
		s.cancel()
	}
	u.mu.Unlock()
	u.reap()
}
func owner(ctx context.Context) (entities.MockOwner, error) {
	actor, err := shared.Actor(ctx)
	if err != nil {
		return entities.MockOwner{}, err
	}
	access, err := shared.Access(ctx)
	if err != nil {
		return entities.MockOwner{}, err
	}
	switch access.Role {
	case "viewer", "editor", "admin", "platform_admin":
	default:
		return entities.MockOwner{}, errs.Forbidden("mock.viewer_required", "Workspace Viewer role is required")
	}
	return entities.MockOwner{ActorID: actor.User.ID, WorkspaceID: access.Workspace.ID}, nil
}
func (u *UseCase) Capabilities(ctx context.Context) (json.RawMessage, error) {
	if _, err := owner(ctx); err != nil {
		return nil, err
	}
	raw, err := u.gateway.Capabilities(ctx)
	if err != nil {
		return json.RawMessage(`{"available":false,"canRun":false,"reason":"mock_unavailable"}`), nil
	}
	var result map[string]any
	if err = json.Unmarshal(raw, &result); err != nil {
		return nil, errs.New("mock.invalid_response", "Invalid capabilities response", 502)
	}
	result["canRun"] = true
	result["idleTimeoutMs"] = minLimit(result["idleTimeoutMs"], int(u.config.IdleTimeout.Milliseconds()))
	result["readyTimeoutMs"] = minLimit(result["readyTimeoutMs"], int(u.config.ReadyTimeout.Milliseconds()))
	result["writeTimeoutMs"] = minLimit(result["writeTimeoutMs"], int(u.config.WriteTimeout.Milliseconds()))
	if l, ok := result["limits"].(map[string]any); ok {
		l["requestBytes"] = minLimit(l["requestBytes"], u.config.RequestBytes)
		l["sessions"] = minLimit(l["sessions"], u.config.Sessions)
		l["sessionsPerOwner"] = minLimit(l["sessionsPerOwner"], u.config.PerOwner)
		l["bufferBytes"] = minLimit(l["bufferBytes"], int(u.config.BufferBytes))
	}
	return json.Marshal(result)
}
func minLimit(x any, n int) int {
	if f, ok := x.(float64); ok && f > 0 && f < float64(n) {
		return int(f)
	}
	return n
}
func (u *UseCase) Generate(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	o, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	return u.gateway.Generate(ctx, o, raw)
}
func (u *UseCase) Create(ctx context.Context, raw json.RawMessage) (entities.MockStream, error) {
	o, err := owner(ctx)
	if err != nil {
		return entities.MockStream{}, err
	}
	u.mu.Lock()
	total, owned := len(u.sessions), 0
	for a, n := range u.pending {
		total += n
		if a.ActorID == o.ActorID {
			owned += n
		}
	}
	for _, s := range u.sessions {
		if s.owner.ActorID == o.ActorID {
			owned++
		}
	}
	if u.ctx.Err() != nil {
		u.mu.Unlock()
		return entities.MockStream{}, errs.New("mock.unavailable", "Backend is shutting down", 503)
	}
	if total >= u.config.Sessions || owned >= u.config.PerOwner {
		u.mu.Unlock()
		return entities.MockStream{}, errs.New("stream.limit_exceeded", "Session capacity exhausted", 429)
	}
	u.pending[o]++
	u.mu.Unlock()
	defer func() {
		u.mu.Lock()
		u.pending[o]--
		if u.pending[o] == 0 {
			delete(u.pending, o)
		}
		u.mu.Unlock()
	}()
	info, err := u.gateway.Create(ctx, o, raw)
	if err != nil {
		return entities.MockStream{}, err
	}
	c, cancel := context.WithCancel(u.ctx)
	s := &session{owner: o, public: uuid.NewString(), upstream: info.ID, expires: minTime(info.ExpiresAt, time.Now().Add(u.config.ReadyTimeout)), idle: min(u.config.IdleTimeout, time.Duration(info.IdleTimeoutMs)*time.Millisecond), ctx: c, cancel: cancel}
	u.mu.Lock()
	if ctx.Err() != nil || u.ctx.Err() != nil {
		u.mu.Unlock()
		u.finish(s)
		return entities.MockStream{}, context.Canceled
	}
	u.sessions[s.public] = s
	u.mu.Unlock()
	return u.publicInfo(s, info), nil
}
func (u *UseCase) publicInfo(s *session, i entities.MockStream) entities.MockStream {
	s.mu.Lock()
	defer s.mu.Unlock()
	i.ID = s.public
	i.ExpiresAt = s.expires
	i.IdleTimeoutMs = s.idle.Milliseconds()
	i.EventsURL = "/api/v1/mock-data/streams/" + s.public + "/events"
	return i
}
func (u *UseCase) find(ctx context.Context, id string) (*session, error) {
	o, err := owner(ctx)
	if err != nil {
		return nil, err
	}
	u.mu.Lock()
	s := u.sessions[id]
	u.mu.Unlock()
	if s == nil || s.owner != o {
		return nil, errs.NotFound("stream.not_found", "Stream does not exist")
	}
	s.mu.Lock()
	expired := !time.Now().Before(s.expires)
	s.mu.Unlock()
	if expired || s.ctx.Err() != nil {
		u.finish(s)
		return nil, errs.NotFound("stream.not_found", "Stream does not exist")
	}
	return s, nil
}
func (u *UseCase) Get(ctx context.Context, id string) (entities.MockStream, error) {
	s, err := u.find(ctx, id)
	if err != nil {
		return entities.MockStream{}, err
	}
	i, err := u.gateway.Get(ctx, s.owner, s.upstream)
	if err != nil {
		if errs.HTTPStatusOf(err) == 404 {
			u.finish(s)
		}
		return i, err
	}
	return u.publicInfo(s, i), nil
}
func (u *UseCase) Update(ctx context.Context, id string, raw json.RawMessage) (entities.MockStream, error) {
	s, err := u.find(ctx, id)
	if err != nil {
		return entities.MockStream{}, err
	}
	i, err := u.gateway.Update(ctx, s.owner, s.upstream, raw)
	if err != nil {
		return i, err
	}
	return u.publicInfo(s, i), nil
}
func (u *UseCase) KeepAlive(ctx context.Context, id string) (entities.MockStream, error) {
	s, err := u.find(ctx, id)
	if err != nil {
		return entities.MockStream{}, err
	}
	started := time.Now()
	i, err := u.gateway.KeepAlive(ctx, s.owner, s.upstream)
	if err != nil {
		return i, err
	}
	s.mu.Lock()
	if s.ctx.Err() != nil || !time.Now().Before(s.expires) || !s.subscribed {
		s.mu.Unlock()
		u.finish(s)
		return entities.MockStream{}, errs.NotFound("stream.not_found", "Stream has expired")
	}
	s.expires = started.Add(min(u.config.IdleTimeout, time.Duration(i.IdleTimeoutMs)*time.Millisecond))
	s.mu.Unlock()
	return u.publicInfo(s, i), nil
}
func (u *UseCase) Stop(ctx context.Context, id string) error {
	s, err := u.find(ctx, id)
	if err != nil {
		return err
	}
	u.finish(s)
	return nil
}
func (u *UseCase) finish(s *session) {
	s.once.Do(func() {
		s.cancel()
		u.mu.Lock()
		delete(u.sessions, s.public)
		// Keep admission reserved until the bounded upstream cleanup finishes.
		u.pending[s.owner]++
		u.mu.Unlock()
		go func() {
			defer func() {
				u.mu.Lock()
				u.pending[s.owner]--
				if u.pending[s.owner] == 0 {
					delete(u.pending, s.owner)
				}
				u.mu.Unlock()
			}()
			ctx, cancel := context.WithTimeout(context.Background(), u.config.WriteTimeout)
			defer cancel()
			_ = u.gateway.Stop(ctx, s.owner, s.upstream)
		}()
	})
}
func (u *UseCase) reap() {
	u.mu.Lock()
	items := make([]*session, 0, len(u.sessions))
	for _, s := range u.sessions {
		items = append(items, s)
	}
	u.mu.Unlock()
	for _, s := range items {
		s.mu.Lock()
		expired := !time.Now().Before(s.expires)
		if expired {
			s.cancel()
		}
		s.mu.Unlock()
		if expired || s.ctx.Err() != nil {
			u.finish(s)
		}
	}
}

type Subscription struct {
	upstream ports.MockSubscription
	session  *session
	usecase  *UseCase
	ctx      context.Context
	cancel   context.CancelFunc
	detach   func() bool
}

func (s *Subscription) Context() context.Context { return s.ctx }
func (s *Subscription) Close() {
	s.cancel()
	s.detach()
	s.upstream.Close()
	s.usecase.finish(s.session)
}
func (s *Subscription) Recv() (*entities.MockEvent, error) {
	e, err := s.upstream.Recv()
	if err != nil {
		return nil, err
	}
	e.StreamID = s.session.public
	return e, nil
}
func (u *UseCase) Subscribe(ctx context.Context, id string) (*Subscription, error) {
	s, err := u.find(ctx, id)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.subscribed {
		s.mu.Unlock()
		return nil, errs.Conflict("stream.already_subscribed", "Stream already has a subscriber")
	}
	if s.ctx.Err() != nil || !time.Now().Before(s.expires) {
		s.mu.Unlock()
		return nil, errs.NotFound("stream.not_found", "Stream has expired")
	}
	s.subscribed = true
	s.mu.Unlock()
	streamctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	off := context.AfterFunc(s.ctx, cancel)
	started := time.Now()
	rpc, err := u.gateway.Subscribe(streamctx, s.owner, s.upstream)
	if err != nil {
		off()
		cancel()
		u.finish(s)
		return nil, err
	}
	s.mu.Lock()
	s.expires = started.Add(s.idle)
	s.mu.Unlock()
	return &Subscription{upstream: rpc, session: s, usecase: u, ctx: streamctx, cancel: cancel, detach: off}, nil
}
func (u *UseCase) ReserveBytes(n int64) bool {
	if u.buffered.Add(n) > u.config.BufferBytes {
		u.buffered.Add(-n)
		return false
	}
	return true
}
func (u *UseCase) ReleaseBytes(n int64) { u.buffered.Add(-n) }

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func (u *UseCase) WriteTimeout() time.Duration { return u.config.WriteTimeout }
func (u *UseCase) RequestLimit() int           { return u.config.RequestBytes }
func (u *UseCase) Stats() (int, int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.sessions), u.buffered.Load()
}
