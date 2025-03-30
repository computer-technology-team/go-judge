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
			// TODO: make this html
			http.Error(w, "user not found", http.StatusNotFound)

			return
		}

		slog.ErrorContext(ctx, "could not get user from database",
			slog.String("username", username), "error", err)
		http.Error(w, "could not get user from storage", http.StatusInternalServerError)
		return
	}

	err = s.templates.Render(ctx, "profilepage", w, user)
	if err != nil {
		slog.ErrorContext(ctx, "could not render profile",
			slog.String("username", username), "error", err)

		http.Error(w, "could not render profile", http.StatusInternalServerError)
		return
	}
}

// GetMeProfile gets the current user's profile
func (s *servicerImpl) GetMeProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update profile logic
	w.Write([]byte("Get Me Profile"))
}

func (s *servicerImpl) ToggleSuperUser(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Toggle Super user"))
}
