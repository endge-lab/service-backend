package ports

import "context"

// TxManager задаёт транзакционную границу, необходимую use case-слою.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	WithinReadTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
