package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	kitconfig "github.com/endge-lab/service-kit-go/config"
)

type MockGeneratorConfig struct {
	GRPCTarget                              string
	Audience                                string
	RequestTimeout                          time.Duration
	HealthTimeout                           time.Duration
	HealthCacheTTL                          time.Duration
	TLS                                     kitconfig.ServiceTLSConfig
	Sessions, PerOwner, RequestBytes        int
	ReadyTimeout, IdleTimeout, WriteTimeout time.Duration
	BufferBytes                             int64
}

func loadMockGeneratorConfig(base *kitconfig.ServiceConfig) (MockGeneratorConfig, error) {
	config := MockGeneratorConfig{
		GRPCTarget:     strings.TrimSpace(os.Getenv("MOCK_GENERATOR_GRPC_TARGET")),
		Audience:       strings.TrimSpace(os.Getenv("MOCK_GENERATOR_AUDIENCE")),
		RequestTimeout: envDuration("MOCK_GENERATOR_REQUEST_TIMEOUT", 10*time.Second),
		HealthTimeout:  envDuration("MOCK_GENERATOR_HEALTH_TIMEOUT", 2*time.Second),
		HealthCacheTTL: envDuration("MOCK_GENERATOR_HEALTH_CACHE_TTL", 5*time.Second),
		TLS: kitconfig.ServiceTLSConfig{
			Enabled: envBool("MOCK_GENERATOR_TLS_ENABLED", false), CertFile: strings.TrimSpace(os.Getenv("MOCK_GENERATOR_TLS_CERT_FILE")),
			KeyFile: strings.TrimSpace(os.Getenv("MOCK_GENERATOR_TLS_KEY_FILE")), CAFile: strings.TrimSpace(os.Getenv("MOCK_GENERATOR_TLS_CA_FILE")),
			InsecureSkipVerify: envBool("MOCK_GENERATOR_TLS_INSECURE_SKIP_VERIFY", false),
		},
	}
	config.Sessions, config.PerOwner, config.RequestBytes, config.BufferBytes = 32, 5, 2<<20, 64<<20
	for name, target := range map[string]*int{"SESSIONS": &config.Sessions, "SESSIONS_PER_OWNER": &config.PerOwner, "REQUEST_BYTES": &config.RequestBytes} {
		if raw := os.Getenv("MOCK_GATEWAY_" + name); raw != "" {
			n, e := strconv.Atoi(raw)
			if e != nil || n < 1 || n > 1<<30 {
				return config, fmt.Errorf("invalid MOCK_GATEWAY_%s", name)
			}
			*target = n
		}
	}
	if raw := os.Getenv("MOCK_GATEWAY_BUFFER_BYTES"); raw != "" {
		n, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || n < 1 || n > 1<<30 {
			return config, fmt.Errorf("invalid MOCK_GATEWAY_BUFFER_BYTES")
		}
		config.BufferBytes = n
	}
	config.ReadyTimeout = envDuration("MOCK_GATEWAY_READY_TIMEOUT", 30*time.Second)
	config.IdleTimeout = envDuration("MOCK_GATEWAY_IDLE_TIMEOUT", 180*time.Second)
	config.WriteTimeout = envDuration("MOCK_GATEWAY_WRITE_TIMEOUT", 10*time.Second)
	if config.ReadyTimeout <= 0 || config.IdleTimeout <= 0 || config.WriteTimeout <= 0 || config.PerOwner > config.Sessions {
		return config, fmt.Errorf("invalid Mock gateway limits")
	}
	if config.Audience == "" {
		config.Audience = "endge-mock-generator"
	}
	if config.GRPCTarget != "" && base.App.IsProduction() && !config.TLS.Enabled {
		return MockGeneratorConfig{}, fmt.Errorf("Mock Generator TLS is required in production")
	}
	if config.RequestTimeout <= 0 || config.HealthTimeout <= 0 || config.HealthCacheTTL <= 0 {
		return MockGeneratorConfig{}, fmt.Errorf("Mock Generator timeouts must be positive")
	}
	if config.GRPCTarget != "" && base.App.IsProduction() && !base.Identity.Client.Enabled {
		return MockGeneratorConfig{}, fmt.Errorf("service identity client must be enabled when Mock Generator is configured in production")
	}
	return config, nil
}
