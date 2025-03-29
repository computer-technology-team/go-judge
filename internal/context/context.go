package context

import (
	"context"

	"github.com/computer-technology-team/go-judge/internal/storage"
)

// Key for user context
type contextKey string

const UserContextKey = contextKey("user")

func GetUserFromContext(ctx context.Context) (user *storage.User, ok bool) {
	user, ok = ctx.Value(UserContextKey).(*storage.User)
	return
}
