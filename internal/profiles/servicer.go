package profiles

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
)

// servicerImpl is the default implementation of the Handler interface
type servicerImpl struct {
	pool *pgxpool.Pool

	querier storage.Querier

	templates *templates.Templates
}

// NewServicer creates a new instance of the default profile handler
func NewServicer(templates *templates.Templates,
	pool *pgxpool.Pool, querier storage.Querier,
) Servicer {
	return &servicerImpl{
		pool:      pool,
		querier:   querier,
		templates: templates,
	}
}

// GetProfile returns a user's profile
func (s *servicerImpl) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := chi.URLParam(r, "username")

	user, err := s.querier.GetUserByUsername(ctx, s.pool, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			templates.RenderError(ctx, w, "user not found", http.StatusNotFound, s.templates)
			return
		}

		slog.ErrorContext(ctx, "could not get user from database",
			slog.String("username", username), "error", err)
		templates.RenderError(ctx, w, "could not get user from storage", http.StatusInternalServerError, s.templates)
		return
	}

	subs, err := s.querier.GetUserSubmissions(ctx, s.pool, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "could not get submissions", slog.String("username", username), "error", err)
		templates.RenderError(ctx, w, "could not retrieve submissions", http.StatusInternalServerError, s.templates)
		return
	}

	model := struct {
		User        storage.User // your SQLC User type
		Submissions []storage.GetUserSubmissionsRow
	}{
		User:        user,
		Submissions: subs[:min(len(subs), 5)],
	}

	err = s.templates.Render(ctx, "profilepage", w, model)
	if err != nil {
		slog.ErrorContext(ctx, "could not render profile",
			slog.String("username", username), "error", err)

		templates.RenderError(ctx, w, "could not render profile", http.StatusInternalServerError, s.templates)
		return
	}
}

func (s *servicerImpl) ToggleSuperUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	username := chi.URLParam(r, "username")

	user, err := s.querier.GetUserByUsername(ctx, s.pool, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			templates.RenderError(ctx, w, "user not found", http.StatusNotFound, s.templates)

			return
		}

		slog.ErrorContext(ctx, "could not get user from database",
			slog.String("username", username), "error", err)
		templates.RenderError(ctx, w, "could not get user from storage", http.StatusInternalServerError, s.templates)
		return
	}

	updatedUser, err := s.querier.ToggleUserSuperLevel(ctx, s.pool, user.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		templates.RenderError(ctx, w, "could not toggle user superuser", http.StatusInternalServerError, s.templates)
		return
	}

	err = s.templates.Render(ctx, "profilepage", w, updatedUser)
	if err != nil {
		slog.ErrorContext(ctx, "could not render profile",
			slog.String("username", user.Username), "error", err)

		templates.RenderError(ctx, w, "could not render profile", http.StatusInternalServerError, s.templates)
		return
	}
}
