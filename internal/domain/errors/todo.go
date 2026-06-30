package errors

var (
	ErrInvalidTodoTitle = InvalidInput("todo.invalid_title", "Некорректный заголовок задачи")
	ErrTodoNotFound     = NotFound("todo.not_found", "Задача не найдена")
)
