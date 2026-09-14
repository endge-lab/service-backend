package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"testing"
)

type nestedTransaction struct{ pgx.Tx }

func TestNestedTransactionJoinsOuterAndPreservesError(t *testing.T) {
	transaction := &nestedTransaction{}
	ctx := context.WithValue(t.Context(), txContextKey{}, transaction)
	wanted := errors.New("release failed")
	calls := 0
	err := (&TxManager{}).WithinTransaction(ctx, func(inner context.Context) error {
		calls++
		got, ok := txFromContext(inner)
		if !ok || got != transaction {
			t.Fatal("nested use case changed transaction")
		}
		return wanted
	})
	if calls != 1 || !errors.Is(err, wanted) {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
}
