package ports

import "context"

// TxManager defines the transaction boundary required by use cases.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
