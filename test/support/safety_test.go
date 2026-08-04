//go:build integration

package support

import (
	"context"
	"testing"
)

// TestDatabaseSafetyGuardRejectsWrongIdentity проверяет fail-closed защиту destructive-операций.
func TestDatabaseSafetyGuardRejectsWrongIdentity(t *testing.T) {
	suite, err := StartPostgresSuite(context.Background())
	if err != nil {
		t.Fatalf("запустить PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := suite.Close(context.Background()); closeErr != nil {
			t.Errorf("закрыть PostgreSQL suite: %v", closeErr)
		}
	})
	database := suite.NewDatabase(t)

	originalName, originalToken := database.Name, database.guardToken
	database.Name = "service_backend"
	if err = database.AssertSafe(context.Background()); err == nil {
		t.Fatal("guard разрешил использовать БД без test-prefix")
	}
	database.Name = originalName
	database.guardToken = "wrong-marker"
	if err = database.AssertSafe(context.Background()); err == nil {
		t.Fatal("guard разрешил использовать БД с неверным marker")
	}
	database.guardToken = originalToken
}
