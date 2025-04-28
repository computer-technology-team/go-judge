-- name: GetUser :one
SELECT *
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username = $1;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, superuser)
VALUES ($1, $2, false)
RETURNING id, username, password_hash, superuser;

-- name: CreateAdmin :one
INSERT INTO users (username, password_hash, superuser)
VALUES ($1, $2, true)
RETURNING id, username, password_hash, superuser;

-- name: ToggleUserSuperLevel :one
UPDATE users
SET superuser = NOT superuser
WHERE id = $1
RETURNING *;

-- name: IncreaseUserAttempts :exec
UPDATE users
SET problems_attempted = problems_attempted + 1
WHERE id = $1;

-- name: IncreaseUserSolves :exec
UPDATE users
SET problems_solved = problems_solved + 1
WHERE id = $1;
