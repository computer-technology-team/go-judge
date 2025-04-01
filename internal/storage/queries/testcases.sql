-- name: InsertTestCase :one
INSERT INTO test_cases (problem_id, input, output)
VALUES ($1, $2, $3)
RETURNING *;
