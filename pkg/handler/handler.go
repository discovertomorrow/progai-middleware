package handler

import (
	"net/http"

	"github.com/discovertomorrow/progai-middleware/pkg/logging"
)

type Endpoint struct {
	Endpoint string
	Parallel int
}

// Limiter creates a middleware that limits the number of concurrent requests
// allowed.
//
// This function returns a middleware function that wraps around an
// http.Handler.
func Limiter(concurrent int) func(http.Handler) http.Handler {
	semaphore := make(chan struct{}, concurrent)

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
			}()
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewDefaultHandler(
	lineByLine bool,
	endpoint Endpoint,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logging.FromContext(ctx).With("function", "handler.<request handler>")
		l.Info("Start handeling request")

		w.Header().Set("Content-Type", "application/json")

		flusher, ok := w.(http.Flusher)
		if !ok {
			l.Warn("ResponseWriter does not support Flusher.")
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint.Endpoint, r.Body)
		if err != nil {
			l.Error("Error creating request", err)
			http.Error(w, "Error creating request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		RequestBackend(
			req,
			func(line []byte) bool {
				_, err := w.Write(line)
				if err != nil {
					l.Info("Error writing line", err)
					return false
				}
				flusher.Flush()
				return true
			},
			lineByLine,
		)

		if err != nil {
			l.Info("Error handeling request", err)
			http.Error(w, "Error handeling request", http.StatusInternalServerError)
			return
		}
		l.Info("Finished Response")
	})
}
