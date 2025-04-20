package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	authenticatorPkg "github.com/computer-technology-team/go-judge/internal/auth/authenticator"
	internalcontext "github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
)

func NewAuthMiddleWare(authenticator authenticatorPkg.Authenticator, pool *pgxpool.Pool, querier storage.Querier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authToken, err := r.Cookie(authenticatorPkg.TokenCookieKey)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := authenticator.VerifyDecodeToken(r.Context(), authToken.Value)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			userUUID, err := uuid.Parse(claims.UserID)
			if err != nil {
				slog.ErrorContext(ctx, "invalid user id in valid token",
					slog.String("token", authToken.Value),
					slog.String("claims.user_id", claims.UserID))
				http.Error(w, "invalid token payload", http.StatusInternalServerError)
				return
			}

			user, err := querier.GetUser(ctx, pool, pgtype.UUID{Valid: true, Bytes: userUUID})
			if err != nil {
				slog.ErrorContext(ctx, "could not get user from database",
					slog.String("claims.user_id", claims.UserID), "userUUID", userUUID)
				http.Error(w, "invalid token payload", http.StatusInternalServerError)
			}

			ctx = context.WithValue(r.Context(), internalcontext.UserContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewRequireAuthMiddleware(tmpl *templates.Templates) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(internalcontext.UserContextKey) == nil {
				renderUnAuthenticated(r.Context(), tmpl, w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func NewRequireSuperUserMiddleware(tmpl *templates.Templates) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, ok := internalcontext.GetUserFromContext(r.Context())
			if !ok {
				renderUnAuthenticated(ctx, tmpl, w)
				return
			}
			if !user.Superuser {
				renderUnAuthorized(ctx, tmpl, w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func renderUnAuthenticated(ctx context.Context, tmpl *templates.Templates, w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	err := tmpl.Render(ctx, "unauthenticated", w, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to render unauthenticated template", "error", err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
}

func renderUnAuthorized(ctx context.Context, tmpl *templates.Templates, w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	err := tmpl.Render(ctx, "unauthorized", w, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to render  template", "error", err)
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
}
