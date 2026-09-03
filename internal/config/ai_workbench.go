package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	kitconfig "github.com/endge-lab/service-kit-go/config"
)

type AIWorkbenchConfig struct {
	GRPCTarget     string
	RequestTimeout time.Duration
	HealthTimeout  time.Duration
	HealthCacheTTL time.Duration
	TLS            kitconfig.ServiceTLSConfig
}

func loadAIWorkbenchConfig(base *kitconfig.ServiceConfig) (AIWorkbenchConfig, error) {
	config := AIWorkbenchConfig{
		GRPCTarget:     strings.TrimSpace(os.Getenv("AI_WORKBENCH_GRPC_TARGET")),
		RequestTimeout: envDuration("AI_WORKBENCH_REQUEST_TIMEOUT", 2*time.Minute),
		HealthTimeout:  envDuration("AI_WORKBENCH_HEALTH_TIMEOUT", 2*time.Second),
		HealthCacheTTL: envDuration("AI_WORKBENCH_HEALTH_CACHE_TTL", 5*time.Second),
		TLS: kitconfig.ServiceTLSConfig{
			Enabled: envBool("AI_WORKBENCH_TLS_ENABLED", false), CertFile: strings.TrimSpace(os.Getenv("AI_WORKBENCH_TLS_CERT_FILE")),
			KeyFile: strings.TrimSpace(os.Getenv("AI_WORKBENCH_TLS_KEY_FILE")), CAFile: strings.TrimSpace(os.Getenv("AI_WORKBENCH_TLS_CA_FILE")),
			InsecureSkipVerify: envBool("AI_WORKBENCH_TLS_INSECURE_SKIP_VERIFY", false),
		},
	}
	if config.RequestTimeout <= 0 || config.HealthTimeout <= 0 || config.HealthCacheTTL <= 0 {
		return AIWorkbenchConfig{}, fmt.Errorf("AI Workbench timeouts must be positive")
	}
	if config.GRPCTarget != "" && base.App.IsProduction() && !base.Identity.Client.Enabled {
		return AIWorkbenchConfig{}, fmt.Errorf("service identity client must be enabled when AI Workbench is configured in production")
	}
	return config, nil
}
