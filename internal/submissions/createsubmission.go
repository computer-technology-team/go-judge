package submissions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	internalcontext "github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/jackc/pgx/v5"
)

const maxFileSize = 10_000_000 // 10MB

// CreateSubmission creates a new submission
func (s *ServicerImpl) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("function", "CreateSubmission", "package", "submissions")
	ctx := r.Context()

	err := r.ParseMultipartForm(maxFileSize)
	if err != nil {
		templates.RenderError(ctx, w, "could not parse form", http.StatusBadRequest, s.templates)
		return
	}

	user, _ := internalcontext.GetUserFromContext(ctx)

	// Parse form data
	problemIDStr := r.PostFormValue("problem_id")
	code := r.PostFormValue("code")
	file, header, err := r.FormFile("file")
	if err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			logger.Error("could not read submission file", "error", err)
			templates.RenderError(ctx, w, "could not read submission file", http.StatusInternalServerError, s.templates)
			return
		}
	} else if code != "" {
		templates.RenderError(ctx, w, "code and file can not be non empty at the same time", http.StatusBadRequest, s.templates)
		return
	} else {
		if header.Size > maxFileSize {
			templates.RenderError(ctx, w, fmt.Sprintf("file is too large (max size: %d bytes)", maxFileSize), http.StatusBadRequest, s.templates)
			return
		}
		codeBytes, err := io.ReadAll(file)
		if err != nil {
			logger.Error("could not read submission file", "error", err)
			templates.RenderError(ctx, w, "could not read submission file", http.StatusInternalServerError, s.templates)
			return
		}

		code = string(codeBytes)
	}

	// Validate form data
	if code == "" {
		templates.RenderError(ctx, w, "solution code is required", http.StatusBadRequest, s.templates)
		return
	}

	problemID, err := strconv.Atoi(problemIDStr)
	if err != nil {
		logger.WarnContext(ctx, "problem id is invalid", "error", err,
			"problem_id", problemIDStr)
		templates.RenderError(ctx, w, "problem id is invalid", http.StatusBadRequest, s.templates)
		return
	}

	logger = logger.With("problem_id", problemID, "user_id", user.ID)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "could not begin transaction", "error", err)
		templates.RenderError(ctx, w, "could not process submission", http.StatusInternalServerError, s.templates)
		return
	}

	defer func(ctx context.Context, tx pgx.Tx) {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			logger.ErrorContext(ctx, "could not rollback transaction", "error", err)
		}
	}(ctx, tx)

	submissionParams := storage.CreateSubmissionParams{
		ProblemID:    int32(problemID),
		UserID:       user.ID,
		SolutionCode: code,
	}

	submission, err := s.querier.CreateSubmission(ctx, tx, submissionParams)
	if err != nil {
		logger.ErrorContext(ctx, "could not create submission", "error", err)
		templates.RenderError(ctx, w, "could not create submission", http.StatusInternalServerError, s.templates)
		return
	}

	err = s.querier.IncreaseUserAttempts(ctx, tx, user.ID)
	if err != nil {
		logger.ErrorContext(ctx, "could not increase user attempts", "error", err)
		templates.RenderError(ctx, w, "could not process submission", http.StatusInternalServerError, s.templates)
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "could not commit transaction", "error", err)
		templates.RenderError(ctx, w, "could not finalize submission", http.StatusInternalServerError, s.templates)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/submissions/%s", submission.ID), http.StatusMovedPermanently)

	go s.broker.AddSubmissionEvaluation(submission)
}
