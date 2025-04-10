-- name: GetAllProblemsSorted :many
SELECT *
FROM problems
ORDER BY created_at DESC;

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
