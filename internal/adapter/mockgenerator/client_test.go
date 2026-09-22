package mockgenerator

import (
	"context"
	"encoding/json"
	pb "github.com/endge-lab/service-backend/internal/adapter/mockpb"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	errs "github.com/endge-lab/service-backend/internal/domain/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type serverFixture struct {
	pb.UnimplementedMockDataServiceServer
	calls   atomic.Int64
	offline atomic.Bool
}

func (s *serverFixture) GetServiceInfo(context.Context, *pb.Empty) (*pb.ServiceInfo, error) {
	s.calls.Add(1)
	if s.offline.Load() {
		return nil, status.Error(codes.Unavailable, "offline")
	}
	return &pb.ServiceInfo{Version: "0.1.0", Env: "test"}, nil
}
func (*serverFixture) Generate(ctx context.Context, _ *pb.GenerateRequest) (*pb.JSONResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (*serverFixture) SubscribeStream(ref *pb.StreamReference, out grpc.ServerStreamingServer[pb.StreamEvent]) error {
	if ref.Owner.ActorId != "actor" || ref.Owner.WorkspaceId != "workspace" {
		return status.Error(codes.PermissionDenied, "invalid context")
	}
	if e := out.Send(&pb.StreamEvent{Json: []byte(`{"type":"started","streamId":"upstream"}`)}); e != nil {
		return e
	}
	select {
	case <-out.Context().Done():
		return out.Context().Err()
	case <-time.After(150 * time.Millisecond):
	}
	return out.Send(&pb.StreamEvent{Json: []byte(`{"type":"data","sequence":1,"items":[9007199254740993]}`)})
}
func TestCacheDeadlineAndLongSubscription(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	fixture := &serverFixture{}
	server := grpc.NewServer()
	pb.RegisterMockDataServiceServer(server, fixture)
	go server.Serve(listener)
	defer server.Stop()
	conn, e := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	client := NewClient(pb.NewMockDataServiceClient(conn), 50*time.Millisecond, time.Second, 40*time.Millisecond)
	ctx := context.Background()
	for range 3 {
		if _, e := client.ServiceInfo(ctx); e != nil {
			t.Fatal(e)
		}
	}
	if fixture.calls.Load() != 1 {
		t.Fatal("successful metadata not cached")
	}
	time.Sleep(50 * time.Millisecond)
	fixture.offline.Store(true)
	for range 3 {
		if _, e := client.ServiceInfo(ctx); errs.HTTPStatusOf(e) != 503 {
			t.Fatal("offline result", e)
		}
	}
	if fixture.calls.Load() != 2 {
		t.Fatal("failed metadata not cached")
	}
	time.Sleep(50 * time.Millisecond)
	fixture.offline.Store(false)
	if _, e := client.ServiceInfo(ctx); e != nil {
		t.Fatal("cache did not recover", e)
	}
	owner := entities.MockOwner{ActorID: "actor", WorkspaceID: "workspace"}
	if _, e := client.Generate(ctx, owner, nil); errs.HTTPStatusOf(e) != 504 {
		t.Fatal("unary deadline", e)
	}
	sub, e := client.Subscribe(ctx, owner, "upstream")
	if e != nil {
		t.Fatal(e)
	}
	defer sub.Close()
	if e, _ := sub.Recv(); e.Type != "started" {
		t.Fatal("missing acknowledgement")
	}
	event, e := sub.Recv()
	if e != nil {
		t.Fatal("stream inherited unary timeout", e)
	}
	raw, _ := json.Marshal(event.Items)
	if string(raw) != "[9007199254740993]" {
		t.Fatal("JSON number changed", string(raw))
	}
}
