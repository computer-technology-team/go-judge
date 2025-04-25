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

-- name: GetUserSubmissions :many
SELECT
    problems.title AS problem_name,
    sqlc.embed(submissions)
FROM submissions INNER JOIN problems ON submissions.problem_id = problems.id
WHERE submissions.user_id = $1;

-- name: GetSubmissionForUser :one
SELECT
    problems.title AS problem_name,
    sqlc.embed(submissions)
FROM submissions INNER JOIN problems ON submissions.problem_id = problems.id
WHERE submissions.user_id = $1 AND submissions.id = $2;
