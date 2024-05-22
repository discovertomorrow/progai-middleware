package session

import (
	"context"
	"net/http"

	"github.com/discovertomorrow/progai-middleware/pkg/logging"
)

type SessionData struct {
	TokenID               int
	UserID                string
	TokenConcurrencyLimit int
}

type key int

const tokenKey key = 0

func FromContext(ctx context.Context) (SessionData, bool) {
	t, ok := ctx.Value(tokenKey).(SessionData)
	return t, ok
}

func SessionIdFromContext(ctx context.Context) int {
	// zero value for token is used if no token
	t, _ := FromContext(ctx)
	return t.TokenID
}

func WithToken(ctx context.Context, s SessionData) context.Context {
	return context.WithValue(ctx, tokenKey, s)
}

func Middleware(getSessionData func(*http.Request) (SessionData, bool)) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l := logging.FromContext(ctx)
			s, ok := getSessionData(r)
			if !ok {
				http.Error(w, "not authorized", http.StatusUnauthorized)
				return
			}
			ctx = WithToken(ctx, s)
			ctx = logging.WithLogger(ctx, l.With(
				"tokenID", s.TokenID, "userID", s.UserID,
				"tokenConcurrencyLimit", s.TokenConcurrencyLimit,
			))
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
