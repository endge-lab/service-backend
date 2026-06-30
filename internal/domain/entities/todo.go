package entities

import "time"

// Todo описывает доменную задачу шаблонного сервиса.
type Todo struct {
	ID          string
	Title       string
	IsCompleted bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
