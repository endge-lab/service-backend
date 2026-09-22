package config

import "testing"

func TestBridgeRequiresExactOriginsAndRejectsProductionDebug(t *testing.T) {
	t.Setenv("BRIDGE_ENABLED", "true")
	t.Setenv("BRIDGE_DEBUG_ENABLED", "true")
	t.Setenv("BRIDGE_ALLOWED_ORIGINS", "https://config.example.com,http://localhost:5173")
	if _, err := loadBridgeConfig(true); err == nil {
		t.Fatal("production accepted debug")
	}
	if _, err := loadBridgeConfig(false); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"", "*", "https://*.example.com", "https://config.example.com/path", "https://user:secret@example.com"} {
		t.Setenv("BRIDGE_ALLOWED_ORIGINS", origin)
		if _, err := loadBridgeConfig(false); err == nil {
			t.Fatalf("accepted invalid origin %q", origin)
		}
	}
	t.Setenv("BRIDGE_ALLOWED_ORIGINS", "https://config.example.com")
	t.Setenv("BRIDGE_DEBUG_ENABLED", "false")
	if _, err := loadBridgeConfig(true); err != nil {
		t.Fatalf("production presence unavailable: %v", err)
	}
}
