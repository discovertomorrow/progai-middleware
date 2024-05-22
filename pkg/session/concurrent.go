package session

import (
	"context"
	"net/http"
	"sync"
)

// TokenLimiter creates a middleware that limits the number of concurrent
// requests allowed per token. It is designed to be used after the main session
// middleware. This middleware utilizes the session middleware's functionality
// to store and retrieve the token and its associated limit within the context.
// It ensures that each token does not exceed its predefined request limit,
// thus helping to control the load on the server.
//
// This function returns a middleware function that wraps around an http.Handler.
func TokenLimiter() func(http.Handler) http.Handler {
	var mutex sync.Mutex
	semaphores := make(map[int]chan struct{}) // concurrent requests per user/token

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			sem, ok := getSemaphore(ctx, &mutex, semaphores)
			if !ok {
				http.Error(w, "no token found", http.StatusInternalServerError)
				return
			}
			sem <- struct{}{}
			defer func() {
				<-sem
			}()
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getSemaphore(
	ctx context.Context,
	mutex *sync.Mutex,
	semaphores map[int]chan struct{},
) (chan struct{}, bool) {
	t, ok := FromContext(ctx)
	if !ok {
		return nil, false
	}
	mutex.Lock()
	defer mutex.Unlock()

	sem, exists := semaphores[t.TokenID]
	if !exists {
		sem = make(chan struct{}, t.TokenConcurrencyLimit)
		semaphores[t.TokenID] = sem
	}
	return sem, true
}
