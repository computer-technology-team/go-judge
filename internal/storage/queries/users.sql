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
