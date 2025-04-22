-- name: InsertTestCase :one
INSERT INTO test_cases (problem_id, input, output)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteProblemTestCases :exec
DELETE FROM test_cases
WHERE problem_id = $1;

-- name: GetTestCasesByProblemID :many
SELECT *
FROM test_cases
WHERE problem_id = $1;

