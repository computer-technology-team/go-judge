package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/computer-technology-team/go-judge/internal/auth"
	internalcontext "github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
)

func NewAuthMiddleWare(authenticator auth.Authenticator, pool *pgxpool.Pool, querier storage.Querier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authToken, err := r.Cookie(auth.TokenCookieKey)
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

// RequireAuth middleware ensures the user is authenticated
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(internalcontext.UserContextKey) == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSuperUser middleware ensures the user is superuser
func RequireSuperUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := internalcontext.GetUserFromContext(r.Context())
		if !ok || !user.Superuser {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
