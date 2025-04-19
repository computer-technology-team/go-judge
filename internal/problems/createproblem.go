package problems

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	internalcontext "github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/jackc/pgx/v5"
)

func (h *DefaultHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := r.ParseForm()
	if err != nil {
		slog.Error("could not parse form data", "error", err)
		templates.RenderError(r.Context(), w, "invalid form data", http.StatusBadRequest, h.templates)
		return
	}

	// Extract form data
	title := r.PostFormValue("title")
	description := r.PostFormValue("description")
	sampleInput := r.PostFormValue("sample_input")
	sampleOutput := r.PostFormValue("sample_output")
	timeLimit := r.PostFormValue("time_limit")
	memoryLimit := r.PostFormValue("memory_limit")
	testCases := []storage.TestCase{}

	for i := 1; ; i++ {
		testInput := r.FormValue("test_input_" + strconv.Itoa(i))
		testOutput := r.FormValue("test_output_" + strconv.Itoa(i))
		if testInput == "" || testOutput == "" {
			break
		}
		testCases = append(testCases, storage.TestCase{
			Input:  testInput,
			Output: testOutput,
		})
	}

	// Validate required fields
	if title == "" || description == "" || sampleInput == "" || sampleOutput == "" {
		slog.Error("missing required fields")
		http.Error(w, "title, description, sample input, and sample output are required", http.StatusBadRequest)
		return
	}

	// Validate at least one test case exists
	if len(testCases) == 0 {
		slog.Error("no test cases provided")
		http.Error(w, "at least one test case is required", http.StatusBadRequest)
		return
	}

	// Convert and validate timeLimit
	timeLimitInt, err := strconv.Atoi(timeLimit)
	if err != nil || timeLimitInt <= 0 {
		slog.Error("invalid time limit", "error", err)
		http.Error(w, "invalid or missing time limit", http.StatusBadRequest)
		return
	}

	// Convert and validate memoryLimit
	memoryLimitInt, err := strconv.Atoi(memoryLimit)
	if err != nil || memoryLimitInt <= 0 {
		slog.Error("invalid memory limit", "error", err)
		http.Error(w, "invalid or missing memory limit", http.StatusBadRequest)
		return
	}

	created_by, _ := internalcontext.GetUserFromContext(r.Context())

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		slog.Error("could not begin transaction", "error", err)
		http.Error(w, "could not start saving", http.StatusInternalServerError)
		return
	}

	defer func(ctx context.Context, tx pgx.Tx) {
		err := tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("could not revert transaction", "error", err)
		}
	}(ctx, tx)

	// Insert the problem into the database
	p, err := h.querier.InsertProblem(ctx, tx, storage.InsertProblemParams{
		Title:         title,
		Description:   description,
		SampleInput:   sampleInput,
		SampleOutput:  sampleOutput,
		TimeLimitMs:   int64(timeLimitInt),
		MemoryLimitKb: int64(memoryLimitInt),
		CreatedBy:     created_by.ID,
	})
	if err != nil {
		slog.Error("could not insert problem", "error", err)
		http.Error(w, "could not insert problem", http.StatusInternalServerError)
		return
	}

	// Insert all test cases into the database
	for _, testCase := range testCases {
		_, err = h.querier.InsertTestCase(ctx, tx, storage.InsertTestCaseParams{
			ProblemID: p.ID,
			Input:     testCase.Input,
			Output:    testCase.Output,
		})
		if err != nil {
			slog.Error("could not insert test case", "error", err)
			http.Error(w, "could not insert test case", http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		slog.Error("could not commit transaction", "error", err)
		http.Error(w, "could not finalize save", http.StatusInternalServerError)
		return
	}

	slog.Info("Problem created successfully", "problem_id", p.ID)

	http.Redirect(w, r, "/problems/"+strconv.Itoa(int(p.ID)), http.StatusSeeOther)
}
