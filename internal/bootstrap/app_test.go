package bootstrap

import (
	"testing"

	"go.uber.org/fx"
)

func TestApplicationDependencyGraph(t *testing.T) {
	err := fx.ValidateApp(appOptions()...)
	if err != nil {
		t.Fatalf("invalid application dependency graph: %v", err)
	}
}
