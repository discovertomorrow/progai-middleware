package usage

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

// UsageTracker creates a middleware function that tracks and updates usage
// metrics throughout the lifecycle of an HTTP request. The middleware
// intercepts HTTP requests and responses, allowing the UsageUpdater to extract
// usage data from the request body and continually update these metrics during
// request processing. After the response is sent, the provided processUsage
// function is called with the final usage metrics, enabling further processing
// or logging of this data.
func UsageTracker(
	updater UsageUpdater,
	processUsage func(context.Context, Usage),
) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading body", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			usage := updater.UsageFromInput(ctx, body)
			pw := passthroughWriter{
				w: w,
				update: func(line string) {
					updater.Update(ctx, usage, line)
				},
				lb: make([]byte, 262144),
			}

			h.ServeHTTP(pw, r)
			processUsage(ctx, *usage)
		})
	}
}
