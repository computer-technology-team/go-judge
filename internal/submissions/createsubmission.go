package submissions

import (
	internalcontext "github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"log/slog"
	"net/http"
	"strconv"
)

// CreateSubmission creates a new submission
func (s *ServicerImpl) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("function", "CreateSubmission", "package", "submissions")
	ctx := r.Context()

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "could not parse form", http.StatusBadRequest)
		return
	}

	user, _ := internalcontext.GetUserFromContext(ctx)

	// Parse form data
	problemIDStr := r.FormValue("problem_id")
	code := r.FormValue("code")

	// Validate form data
	if code == "" {
		http.Error(w, "solution code is required", http.StatusBadRequest)
		return
	}

	problemID, err := strconv.Atoi(problemIDStr)
	if err != nil {
		logger.WarnContext(ctx, "problem id is invalid", "error", err,
			"problem_id", problemIDStr)
		http.Error(w, "problem id is invalid", http.StatusBadRequest)
		return
	}

	logger = logger.With("problem_id", problemID, "user_id", user.ID)

	// Create submission in database
	submissionParams := storage.CreateSubmissionParams{
		ProblemID:    int32(problemID),
		UserID:       user.ID,
		SolutionCode: code,
	}

	submission, err := s.querier.CreateSubmission(ctx, s.pool, submissionParams)
	if err != nil {
		logger.ErrorContext(ctx, "could not create submission", "error", err)
		http.Error(w, "could not create submission", http.StatusInternalServerError)
		return
	}

	go s.broker.AddSubmissionEvaluation(submission)
}
