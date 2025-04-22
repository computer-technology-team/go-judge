-- name: GetAllProblemsSorted :many
SELECT problems.*, users.username as author_name
FROM problems JOIN users  ON problems.created_by = users.id
ORDER BY published_at DESC
LIMIT $1
OFFSET $2;

-- name: GetUserProblemsSorted :many
SELECT *
FROM problems
WHERE created_by = $3
ORDER BY published_at DESC
LIMIT $1
OFFSET $2;

-- name: GetAllPublishedProblemsSorted :many
SELECT *
FROM problems
WHERE draft = false
ORDER BY published_at DESC
LIMIT $1
OFFSET $2;

-- name: GetProblemByID :one
SELECT *
FROM problems
WHERE id = $1;

-- name: GetProblemForUser :one
SELECT *
FROM problems
WHERE id = $1 and (draft = FALSE or created_by = $2 or sqlc.arg(is_admin)::BOOLEAN);

-- name: PublishProblem :exec
UPDATE problems
SET draft = FALSE, published_at = now()
WHERE id = $1;

-- name: DraftProblem :exec
UPDATE problems
SET draft = TRUE
WHERE id = $1;

-- name: InsertProblem :one
INSERT INTO problems (
    title,
    description,
    sample_input,
    sample_output,
    time_limit_ms,
    memory_limit_kb,
    created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateProblem :one
UPDATE problems
SET
    title = $2,
    description = $3,
    sample_input = $4,
    sample_output = $5,
    time_limit_ms = $6,
    memory_limit_kb = $7
WHERE id = $1
RETURNING *;
