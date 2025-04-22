package problems

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// UpdateProblem updates a specific problem
func (h *DefaultHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		templates.RenderError(ctx, w, "invalid problem id", http.StatusBadRequest, h.templates)
		return
	}

	err = r.ParseForm()
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
		templates.RenderError(r.Context(), w, "title, description, sample input, and sample output are required", http.StatusBadRequest, h.templates)
		return
	}

	// Validate at least one test case exists
	if len(testCases) == 0 {
		slog.Error("no test cases provided")
		templates.RenderError(r.Context(), w, "at least one test case is required", http.StatusBadRequest, h.templates)
		return
	}

	// Convert and validate timeLimit
	timeLimitInt, err := strconv.Atoi(timeLimit)
	if err != nil || timeLimitInt <= 0 {
		slog.Error("invalid time limit", "error", err)
		templates.RenderError(r.Context(), w, "invalid or missing time limit", http.StatusBadRequest, h.templates)
		return
	}

	// Convert and validate memoryLimit
	memoryLimitInt, err := strconv.Atoi(memoryLimit)
	if err != nil || memoryLimitInt <= 0 {
		slog.Error("invalid memory limit", "error", err)
		templates.RenderError(r.Context(), w, "invalid or missing memory limit", http.StatusBadRequest, h.templates)
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		templates.RenderError(ctx, w, "could not begin update", http.StatusInternalServerError, h.templates)
		return
	}

	defer func(ctx context.Context, tx pgx.Tx) {
		err := tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) && err != nil {
			slog.Error("could not rollback", "error", err)
		}
	}(ctx, tx)

	// Delete existing test cases
	err = h.querier.DeleteProblemTestCases(ctx, tx, int32(id))
	if err != nil {
		slog.Error("could not reset testcases", "error", err)
		templates.RenderError(ctx, w, "could not reset testcases", http.StatusInternalServerError, h.templates)
		return
	}

	// Update the problem
	p, err := h.querier.UpdateProblem(ctx, tx, storage.UpdateProblemParams{
		ID:            int32(id),
		Title:         title,
		Description:   description,
		SampleInput:   sampleInput,
		SampleOutput:  sampleOutput,
		TimeLimitMs:   int64(timeLimitInt),
		MemoryLimitKb: int64(memoryLimitInt),
	})
	if err != nil {
		slog.Error("could not update problem", "error", err)
		templates.RenderError(ctx, w, "could not update problem", http.StatusInternalServerError, h.templates)
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
			templates.RenderError(r.Context(), w, "could not insert test case", http.StatusInternalServerError, h.templates)
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		slog.Error("could not commit transaction", "error", err)
		templates.RenderError(r.Context(), w, "could not finalize update", http.StatusInternalServerError, h.templates)
		return
	}

	slog.Info("Problem updated successfully", "problem_id", p.ID)

	http.Redirect(w, r, "/problems/"+strconv.Itoa(int(p.ID)), http.StatusSeeOther)
}
