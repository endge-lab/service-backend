package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// BridgeConfig — локальная политика backend; debug никогда не доступен в production.
type BridgeConfig struct {
	Enabled        bool
	DebugEnabled   bool
	AllowedOrigins []string
}

func loadBridgeConfig(production bool) (BridgeConfig, error) {
	value := BridgeConfig{Enabled: envBool("BRIDGE_ENABLED", false), DebugEnabled: envBool("BRIDGE_DEBUG_ENABLED", false)}
	if production && value.DebugEnabled {
		return value, fmt.Errorf("BRIDGE_DEBUG_ENABLED is forbidden in production")
	}
	for _, raw := range csv(os.Getenv("BRIDGE_ALLOWED_ORIGINS")) {
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || strings.Contains(parsed.Host, "*") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return value, fmt.Errorf("BRIDGE_ALLOWED_ORIGINS must contain exact HTTP(S) origins")
		}
		value.AllowedOrigins = append(value.AllowedOrigins, strings.TrimRight(parsed.String(), "/"))
	}
	if value.Enabled && len(value.AllowedOrigins) == 0 {
		return value, fmt.Errorf("BRIDGE_ALLOWED_ORIGINS is required when BRIDGE_ENABLED=true")
	}
	return value, nil
}
