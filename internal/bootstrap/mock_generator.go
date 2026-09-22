package bootstrap

import (
	"context"

	mockadapter "github.com/endge-lab/service-backend/internal/adapter/mockgenerator"
	mockpb "github.com/endge-lab/service-backend/internal/adapter/mockpb"
	"github.com/endge-lab/service-backend/internal/config"
	"github.com/endge-lab/service-kit-go/pkg/grpckit"
	serviceoidc "github.com/endge-lab/service-kit-go/pkg/oidc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

func newMockGeneratorGateway(lifecycle fx.Lifecycle, cfg *config.Config) (*mockadapter.Client, error) {
	if cfg.MockGenerator.GRPCTarget == "" {
		return mockadapter.NewClient(nil, cfg.MockGenerator.RequestTimeout, cfg.MockGenerator.HealthTimeout, cfg.MockGenerator.HealthCacheTTL), nil
	}
	dialOptions := make([]grpc.DialOption, 0, 2)
	identityConfig := cfg.ServiceConfig.Identity.Client
	if identityConfig.Enabled {
		provider, err := serviceoidc.NewClientCredentialsProvider(serviceoidc.ClientCredentialsConfig{
			TokenURL: identityConfig.TokenURL, ClientID: identityConfig.ClientID, ClientSecret: identityConfig.ClientSecret,
			Audience: cfg.MockGenerator.Audience, Scope: identityConfig.Scope, Timeout: identityConfig.Timeout,
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
		Target: cfg.MockGenerator.GRPCTarget, DefaultTimeout: cfg.MockGenerator.RequestTimeout,
		MaxReceiveBytes: 32 * 1024 * 1024, MaxSendBytes: 32 * 1024 * 1024, Compression: true,
		TLS: grpckit.TLSConfig{
			Enabled: cfg.MockGenerator.TLS.Enabled, CertFile: cfg.MockGenerator.TLS.CertFile, KeyFile: cfg.MockGenerator.TLS.KeyFile,
			CAFile: cfg.MockGenerator.TLS.CAFile, InsecureSkipVerify: cfg.MockGenerator.TLS.InsecureSkipVerify,
		},
	}, dialOptions...)
	if err != nil {
		return nil, err
	}
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error { return connection.Close() }})
	return mockadapter.NewClient(
		mockpb.NewMockDataServiceClient(connection),
		cfg.MockGenerator.RequestTimeout, cfg.MockGenerator.HealthTimeout, cfg.MockGenerator.HealthCacheTTL,
	), nil
}
