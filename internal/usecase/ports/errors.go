package ports

import "errors"

// ErrNotFound is returned by repository adapters when a requested row does
// not exist. Use cases must not depend on database-driver-specific errors.
var ErrNotFound = errors.New("repository entity not found")
