package openai

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/discovertomorrow/progai-middleware/pkg/handler"
	"github.com/discovertomorrow/progai-middleware/pkg/logging"
)

func NewOpenAiChatHandler(
	lineByLine bool,
	endpoint handler.Endpoint,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logging.FromContext(ctx).With("function", "openai.<request handler>")
		l.Info("Start handeling request")

		w.Header().Set("Content-Type", "application/json")

		dec := json.NewDecoder(r.Body)
		var req ChatRequest
		err := dec.Decode(&req)
		if err != nil {
			l.Info("Error unmarshaling Body", err)
			http.Error(w, "error unmarshaling request", http.StatusBadRequest)
			return
		}

		l.With("userRequest", req)

		req.MaxTokens = 500
		req.Model = "/tmp/models/zephyr/"

		buf := bytes.Buffer{}
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(req); err != nil {
			l.Info("Error encoding request", err)
			http.Error(w, "error encoding request", http.StatusInternalServerError)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			l.Warn("ResponseWriter does not support Flusher.")
			return
		}

		backendReq, err := http.NewRequestWithContext(
			ctx,
			"POST",
			endpoint.Endpoint,
			bytes.NewReader(buf.Bytes()),
		)
		if err != nil {
			l.Error("Error creating request", err)
			http.Error(w, "Error creating request", http.StatusInternalServerError)
			return
		}
		backendReq.Header.Set("Content-Type", "application/json")
		backendReq.Header.Set("Authorization", "Bearer sk-example")

		err = handler.RequestBackend(
			backendReq,
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
