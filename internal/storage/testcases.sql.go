// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: testcases.sql

package storage

import (
	"context"
)

const deleteProblemTestCases = `-- name: DeleteProblemTestCases :exec
DELETE FROM test_cases
WHERE problem_id = $1
`

func (q *Queries) DeleteProblemTestCases(ctx context.Context, db DBTX, problemID int32) error {
	_, err := db.Exec(ctx, deleteProblemTestCases, problemID)
	return err
}

const getTestCasesByProblemID = `-- name: GetTestCasesByProblemID :many
SELECT id, problem_id, input, output
FROM test_cases
WHERE problem_id = $1
`

func (q *Queries) GetTestCasesByProblemID(ctx context.Context, db DBTX, problemID int32) ([]TestCase, error) {
	rows, err := db.Query(ctx, getTestCasesByProblemID, problemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TestCase
	for rows.Next() {
		var i TestCase
		if err := rows.Scan(
			&i.ID,
			&i.ProblemID,
			&i.Input,
			&i.Output,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertTestCase = `-- name: InsertTestCase :one
INSERT INTO test_cases (problem_id, input, output)
VALUES ($1, $2, $3)
RETURNING id, problem_id, input, output
`

type InsertTestCaseParams struct {
	ProblemID int32  `db:"problem_id" json:"problem_id"`
	Input     string `db:"input" json:"input"`
	Output    string `db:"output" json:"output"`
}

func (q *Queries) InsertTestCase(ctx context.Context, db DBTX, arg InsertTestCaseParams) (TestCase, error) {
	row := db.QueryRow(ctx, insertTestCase, arg.ProblemID, arg.Input, arg.Output)
	var i TestCase
	err := row.Scan(
		&i.ID,
		&i.ProblemID,
		&i.Input,
		&i.Output,
	)
	return i, err
}
