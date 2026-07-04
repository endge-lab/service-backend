-- name: CreateTodo :one
INSERT INTO todos (
  id,
  title,
  is_completed,
  created_at,
  updated_at
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, title, is_completed, created_at, updated_at;

-- name: GetTodoByID :one
SELECT id, title, is_completed, created_at, updated_at
FROM todos
WHERE id = $1;

-- name: UpdateTodo :one
UPDATE todos
SET
  title = $2,
  is_completed = $3,
  updated_at = $4
WHERE id = $1
RETURNING id, title, is_completed, created_at, updated_at;

-- name: DeleteTodo :execrows
DELETE FROM todos
WHERE id = $1;
