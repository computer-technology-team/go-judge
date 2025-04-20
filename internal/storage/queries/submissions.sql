-- name: CreateSubmission :one
INSERT INTO submissions (problem_id, user_id, solution_code)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateSubmissionStatus :one
UPDATE submissions
SET status = $2, message = $3
WHERE id = $1
RETURNING *;

-- name: RetrySubmissionDueToInternalError :one
UPDATE submissions
SET retries = retries + 1
WHERE id = $1
RETURNING *;
