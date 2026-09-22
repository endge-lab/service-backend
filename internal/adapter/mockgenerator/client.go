package mockgenerator

import (
	"context"
	"encoding/json"
	pb "github.com/endge-lab/service-backend/internal/adapter/mockpb"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	errs "github.com/endge-lab/service-backend/internal/domain/errors"
	"github.com/endge-lab/service-backend/internal/usecase/ports"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	rpc                         pb.MockDataServiceClient
	timeout, healthTimeout, ttl time.Duration
	mu                          sync.Mutex
	at                          time.Time
	info                        ports.ConnectedServiceInfo
	infoErr                     error
}

func NewClient(rpc pb.MockDataServiceClient, timeout, healthTimeout, ttl time.Duration) *Client {
	return &Client{rpc: rpc, timeout: timeout, healthTimeout: healthTimeout, ttl: ttl}
}
func unavailable() error { return errs.New("mock.unavailable", "Mock Generator is unavailable", 503) }
func (c *Client) ServiceInfo(ctx context.Context) (ports.ConnectedServiceInfo, error) {
	fallback := ports.ConnectedServiceInfo{Service: "service_mock_generator"}
	if c.rpc == nil {
		return fallback, unavailable()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.at.IsZero() && time.Since(c.at) < c.ttl {
		return c.info, c.infoErr
	}
	ctx, cancel := context.WithTimeout(ctx, c.healthTimeout)
	defer cancel()
	r, err := c.rpc.GetServiceInfo(ctx, &pb.Empty{})
	c.at = time.Now()
	c.info = fallback
	c.infoErr = mapError(err)
	if err == nil {
		c.info.Version = r.GetVersion()
		c.info.Env = r.GetEnv()
	}
	return c.info, c.infoErr
}
func (c *Client) call(ctx context.Context, fn func(context.Context) (*pb.JSONResponse, error)) (json.RawMessage, error) {
	if c.rpc == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	r, err := fn(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	if !json.Valid(r.GetJson()) {
		return nil, errs.New("mock.invalid_response", "Invalid generator response", 502)
	}
	return json.RawMessage(r.GetJson()), nil
}
func (c *Client) Capabilities(ctx context.Context) (json.RawMessage, error) {
	return c.call(ctx, func(ctx context.Context) (*pb.JSONResponse, error) { return c.rpc.GetCapabilities(ctx, &pb.Empty{}) })
}
func toOwner(o entities.MockOwner) *pb.Owner {
	return &pb.Owner{ActorId: o.ActorID, WorkspaceId: o.WorkspaceID}
}
func ref(o entities.MockOwner, id string) *pb.StreamReference {
	return &pb.StreamReference{Owner: toOwner(o), Id: id}
}
func (c *Client) Generate(ctx context.Context, o entities.MockOwner, raw json.RawMessage) (json.RawMessage, error) {
	return c.call(ctx, func(ctx context.Context) (*pb.JSONResponse, error) {
		return c.rpc.Generate(ctx, &pb.GenerateRequest{Owner: toOwner(o), Json: raw})
	})
}
func decodeInfo(raw json.RawMessage, err error) (entities.MockStream, error) {
	if err != nil {
		return entities.MockStream{}, err
	}
	var info entities.MockStream
	if err = json.Unmarshal(raw, &info); err != nil || info.ID == "" || info.IdleTimeoutMs <= 0 || info.ExpiresAt.IsZero() {
		return info, errs.New("mock.invalid_response", "Invalid stream response", 502)
	}
	return info, nil
}
func (c *Client) Create(ctx context.Context, o entities.MockOwner, raw json.RawMessage) (entities.MockStream, error) {
	return decodeInfo(c.call(ctx, func(ctx context.Context) (*pb.JSONResponse, error) {
		return c.rpc.CreateStream(ctx, &pb.CreateStreamRequest{Owner: toOwner(o), Json: raw})
	}))
}
func (c *Client) Get(ctx context.Context, o entities.MockOwner, id string) (entities.MockStream, error) {
	return decodeInfo(c.call(ctx, func(ctx context.Context) (*pb.JSONResponse, error) { return c.rpc.GetStream(ctx, ref(o, id)) }))
}
func (c *Client) Update(ctx context.Context, o entities.MockOwner, id string, raw json.RawMessage) (entities.MockStream, error) {
	return decodeInfo(c.call(ctx, func(ctx context.Context) (*pb.JSONResponse, error) {
		return c.rpc.UpdateStream(ctx, &pb.UpdateStreamRequest{Stream: ref(o, id), PatchJson: raw})
	}))
}
func (c *Client) KeepAlive(ctx context.Context, o entities.MockOwner, id string) (entities.MockStream, error) {
	return decodeInfo(c.call(ctx, func(ctx context.Context) (*pb.JSONResponse, error) { return c.rpc.KeepAliveStream(ctx, ref(o, id)) }))
}
func (c *Client) Stop(ctx context.Context, o entities.MockOwner, id string) error {
	if c.rpc == nil {
		return unavailable()
	}
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	_, err := c.rpc.StopStream(ctx, ref(o, id))
	return mapError(err)
}

type subscription struct {
	stream grpc.ServerStreamingClient[pb.StreamEvent]
	cancel context.CancelFunc
	first  *entities.MockEvent
}

func (s *subscription) Close() { s.cancel() }
func (s *subscription) Recv() (*entities.MockEvent, error) {
	if s.first != nil {
		x := s.first
		s.first = nil
		return x, nil
	}
	r, err := s.stream.Recv()
	if err != nil {
		return nil, mapError(err)
	}
	var e entities.MockEvent
	if err = json.Unmarshal(r.GetJson(), &e); err != nil || e.Type == "" {
		return nil, errs.New("mock.invalid_response", "Invalid stream event", 502)
	}
	return &e, nil
}
func (c *Client) Subscribe(ctx context.Context, o entities.MockOwner, id string) (ports.MockSubscription, error) {
	if c.rpc == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithCancel(ctx)
	rpc, err := c.rpc.SubscribeStream(ctx, ref(o, id))
	if err != nil {
		cancel()
		return nil, mapError(err)
	}
	s := &subscription{stream: rpc, cancel: cancel}
	type firstResult struct {
		event *entities.MockEvent
		err   error
	}
	done := make(chan firstResult, 1)
	go func() { e, err := s.Recv(); done <- firstResult{e, err} }()
	timer := time.NewTimer(c.timeout)
	defer timer.Stop()
	select {
	case r := <-done:
		if r.err != nil {
			cancel()
			return nil, r.err
		}
		if r.event.Type != "started" {
			cancel()
			return nil, errs.New("mock.invalid_response", "Missing stream acknowledgement", 502)
		}
		s.first = r.event
		return s, nil
	case <-timer.C:
		cancel()
		return nil, errs.New("mock.timeout", "Stream acknowledgement timed out", 504)
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}
}
func mapError(err error) error {
	if err == nil {
		return nil
	}
	s, ok := status.FromError(err)
	if !ok {
		return err
	}
	code := errs.Code("mock.upstream_error")
	message := "Mock Generator request failed"
	httpStatus := 502
	switch s.Code() {
	case codes.InvalidArgument:
		httpStatus = 400
	case codes.NotFound:
		httpStatus = 404
	case codes.AlreadyExists, codes.FailedPrecondition:
		httpStatus = 409
	case codes.ResourceExhausted:
		httpStatus = 429
	case codes.DeadlineExceeded:
		httpStatus = 504
	case codes.Canceled:
		httpStatus = 499
	case codes.Unavailable, codes.Unauthenticated, codes.PermissionDenied:
		httpStatus = 503
	}
	details := map[string]any{}
	for _, d := range s.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info.Domain == "mockdata.v1" {
			code = errs.Code(info.Reason)
			message = s.Message()
			if path := info.Metadata["path"]; path != "" {
				details["path"] = path
			}
			if n, e := strconv.Atoi(info.Metadata["httpStatus"]); e == nil && (n == 413 || n == 429) {
				httpStatus = n
			}
			if info.Reason == "stream.stopped" {
				httpStatus = 200
			}
		}
	}
	return errs.WithDetails(errs.New(code, message, httpStatus), details)
}
