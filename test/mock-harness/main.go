// Isolated loopback transport harness. Synthetic users bypass DB identity resolution;
// production OIDC service credentials, gateway, access policy and SSE handler are used.
package main

import (
	"bytes"
	"context"
	adapter "github.com/endge-lab/service-backend/internal/adapter/mockgenerator"
	pb "github.com/endge-lab/service-backend/internal/adapter/mockpb"
	transport "github.com/endge-lab/service-backend/internal/api/http/v1/mock_data"
	"github.com/endge-lab/service-backend/internal/domain/entities"
	resource "github.com/endge-lab/service-backend/internal/usecase/mock_data"
	"github.com/endge-lab/service-kit-go/pkg/grpckit"
	"github.com/endge-lab/service-kit-go/pkg/oidc"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

func main() {
	provider, e := oidc.NewClientCredentialsProvider(oidc.ClientCredentialsConfig{TokenURL: os.Getenv("HARNESS_TOKEN_URL"), ClientID: "endge-service-backend", ClientSecret: os.Getenv("HARNESS_CLIENT_SECRET"), Audience: "endge-mock-generator", Timeout: 2 * time.Second})
	if e != nil {
		log.Fatal(e)
	}
	conn, e := grpckit.NewClient(grpckit.ClientConfig{Target: "127.0.0.1:" + os.Getenv("HARNESS_GRPC_PORT"), MaxReceiveBytes: 23 << 20, MaxSendBytes: 3 << 20}, grpc.WithChainUnaryInterceptor(grpckit.UnaryClientIdentityInterceptor(provider)), grpc.WithChainStreamInterceptor(grpckit.StreamClientIdentityInterceptor(provider)))
	if e != nil {
		log.Fatal(e)
	}
	defer conn.Close()
	gateway := adapter.NewClient(pb.NewMockDataServiceClient(conn), 10*time.Second, 2*time.Second, 5*time.Second)
	cfg := resource.DefaultConfig()
	if raw := os.Getenv("MOCK_IDLE_TIMEOUT"); raw != "" {
		cfg.IdleTimeout, _ = time.ParseDuration(raw)
	}
	if raw := os.Getenv("MOCK_READY_TIMEOUT"); raw != "" {
		cfg.ReadyTimeout, _ = time.ParseDuration(raw)
	}
	u := resource.NewUseCase(gateway, cfg)
	u.Start()
	defer u.Close()
	app := fiber.New(fiber.Config{DisableStartupMessage: true, BodyLimit: 3 << 20})
	app.Get("/goroutines", func(c *fiber.Ctx) error {
		var b bytes.Buffer
		_ = pprof.Lookup("goroutine").WriteTo(&b, 2)
		return c.Send(b.Bytes())
	})
	app.Get("/stats", func(c *fiber.Ctx) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		n, b := u.Stats()
		return c.JSON(fiber.Map{"sessions": n, "bufferedBytes": b, "heap": m.HeapAlloc, "goroutines": runtime.NumGoroutine()})
	})
	app.Use(func(c *fiber.Ctx) error {
		actor, workspace := strings.Clone(c.Get("X-Test-Actor")), strings.Clone(c.Get("X-Endge-Workspace"))
		if actor == "" || workspace == "" {
			return c.SendStatus(401)
		}
		role := strings.Clone(c.Get("X-Test-Role", "viewer"))
		ctx := entities.WithCurrentActor(context.Background(), entities.CurrentActor{User: &entities.User{ID: actor}})
		ctx = entities.WithWorkspaceAccess(ctx, entities.WorkspaceAccess{Workspace: entities.Workspace{ID: workspace}, Role: role})
		c.SetUserContext(ctx)
		return c.Next()
	})
	transport.RegisterRoutes(app.Group("/api/v1"), transport.NewHandler(transport.BindUseCase(u)))
	log.Fatal(app.Listen("127.0.0.1:" + os.Getenv("HARNESS_HTTP_PORT")))
}
