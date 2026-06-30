package valueobjects

import (
	"strings"

	domainerrors "github.com/endge-lab/service-template-go/internal/domain/errors"
)

const todoTitleMaxLength = 160

type TodoTitle struct {
	value string
}

func NewTodoTitle(value string) (TodoTitle, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return TodoTitle{}, domainerrors.ErrInvalidTodoTitle
	}
	if len([]rune(normalized)) > todoTitleMaxLength {
		return TodoTitle{}, domainerrors.ErrInvalidTodoTitle
	}

	return TodoTitle{value: normalized}, nil
}

func (t TodoTitle) Value() string {
	return t.value
}

func (t TodoTitle) String() string {
	return t.value
}
