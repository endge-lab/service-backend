//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/endge-lab/service-backend/test/support"
)

var postgresSuite *support.PostgresSuite

// TestMain поднимает единственный контейнер package и никогда не принимает внешний DSN.
func TestMain(m *testing.M) {
	var err error
	postgresSuite, err = support.StartPostgresSuite(context.Background())
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "integration suite требует доступный Docker: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if err = postgresSuite.Close(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "не удалось удалить тестовый PostgreSQL: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
