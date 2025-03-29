-- name: GetUser :one
SELECT *
FROM users
WHERE id = $1;


-- name: GetUserByUsername :one
SELECT *
FROM users
WHERE username = $1;
