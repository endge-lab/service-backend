package bootstrap

import (
	"testing"

	"go.uber.org/fx"
)

// TestApplicationDependencyGraph проверяет полноту Fx-графа production-приложения.
func TestApplicationDependencyGraph(t *testing.T) {
	err := fx.ValidateApp(appOptions()...)
	if err != nil {
		t.Fatalf("invalid application dependency graph: %v", err)
	}
}
