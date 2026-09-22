package bootstrap

import (
	"context"

	workbenchadapter "github.com/endge-lab/service-backend/internal/adapter/workbench"
	workbenchpb "github.com/endge-lab/service-backend/internal/adapter/workbenchpb"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-kit-go/pkg/grpckit"
	serviceoidc "github.com/endge-lab/service-kit-go/pkg/oidc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func newAIWorkbenchGateway(lifecycle fx.Lifecycle, cfg *config.Config) (*workbenchadapter.Client, error) {
	if cfg.AIWorkbench.GRPCTarget == "" {
		return workbenchadapter.NewClient(nil, nil, cfg.AIWorkbench.RequestTimeout, cfg.AIWorkbench.HealthTimeout, cfg.AIWorkbench.HealthCacheTTL), nil
	}
	dialOptions := make([]grpc.DialOption, 0, 2)
	identityConfig := cfg.ServiceConfig.Identity.Client
	if identityConfig.Enabled {
		provider, err := serviceoidc.NewClientCredentialsProvider(serviceoidc.ClientCredentialsConfig{
			TokenURL: identityConfig.TokenURL, ClientID: identityConfig.ClientID, ClientSecret: identityConfig.ClientSecret,
			Audience: identityConfig.Audience, Scope: identityConfig.Scope, Timeout: identityConfig.Timeout,
		})
		if err != nil {
			return nil, err
		}
		dialOptions = append(dialOptions,
			grpc.WithChainUnaryInterceptor(grpckit.UnaryClientIdentityInterceptor(provider)),
			grpc.WithChainStreamInterceptor(grpckit.StreamClientIdentityInterceptor(provider)),
		)
	}
	connection, err := grpckit.NewClient(grpckit.ClientConfig{
		Target: cfg.AIWorkbench.GRPCTarget, DefaultTimeout: cfg.AIWorkbench.RequestTimeout,
		MaxReceiveBytes: 32 * 1024 * 1024, MaxSendBytes: 32 * 1024 * 1024, Compression: true,
		TLS: grpckit.TLSConfig{
			Enabled: cfg.AIWorkbench.TLS.Enabled, CertFile: cfg.AIWorkbench.TLS.CertFile, KeyFile: cfg.AIWorkbench.TLS.KeyFile,
			CAFile: cfg.AIWorkbench.TLS.CAFile, InsecureSkipVerify: cfg.AIWorkbench.TLS.InsecureSkipVerify,
		},
	}, dialOptions...)
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error { return connection.Close() }})
	return workbenchadapter.NewClient(
		workbenchpb.NewWorkbenchServiceClient(connection), grpc_health_v1.NewHealthClient(connection),
		cfg.AIWorkbench.RequestTimeout, cfg.AIWorkbench.HealthTimeout, cfg.AIWorkbench.HealthCacheTTL,
	), nil
}
