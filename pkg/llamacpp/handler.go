package llamacpp

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/discovertomorrow/progai-middleware/pkg/handler"
	"github.com/discovertomorrow/progai-middleware/pkg/logging"
	"github.com/discovertomorrow/progai-middleware/pkg/openai"
	"github.com/discovertomorrow/progai-middleware/pkg/session"
)

func NewLlamacppHandler(
	lineByLine bool,
	endpoints []handler.Endpoint,
) http.Handler {
	queue := NewQueue(endpoints)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logging.FromContext(ctx).With("function", "handler.<request handler>")
		l.Info("Start handeling request")

		dec := json.NewDecoder(r.Body)
		var req Request
		err := dec.Decode(&req)
		if err != nil {
			l.Info("Error unmarshaling Body", err)
			http.Error(w, "error unmarshaling request", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			l.Warn("ResponseWriter does not support Flusher.")
			return
		}

		slot := queue.RequestSlot(session.SessionIdFromContext(ctx), req.Slot)
		defer queue.ReleaseSlot(slot)
		l = l.With(
			"slot", slot.ID,
			"userSlot", req.Slot,
			"endpointSlot", slot.endpointSlot.slot,
			"endpoint", slot.endpointSlot.endpoint,
		)
		l.Info("Got slot")

		w.Header().Set("Content-Type", "application/json")

		if err := handleLlamacpp(
			ctx,
			slot,
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
		); err != nil {
			http.Error(w, "Error requesting response", http.StatusInternalServerError)
		}

		l.Info("Finished Response")
	})
}

func NewLlamacppChatHandler(
	logger *slog.Logger,
	lineByLine bool,
	endpoints []handler.Endpoint,
	chatTemplate string,
	stop []string,
) http.Handler {
	logger.Warn("LlamacppChatHandler is experimental")
	queue := NewQueue(endpoints)
	tmpl, err := template.New("chat").Parse(chatTemplate)
	if err != nil {
		logger.Error("Error parsing template", err)
		// we cannot recover from this
		panic(err)
	}
	prepareChatPrompt := func(msgs []openai.Message) (string, error) {
		buf := bytes.Buffer{}
		tmpl.Execute(&buf, msgs)
		if err != nil {
			return "", err
		}
		return buf.String(), err
	}

	toolCalls := NewExpiringMap()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logging.FromContext(ctx).With("function", "handler.<chat request handler>")
		llamacppRequestId := strconv.FormatInt(time.Now().UnixNano(), 16)
		l.Info("Start handeling chat request")

		dec := json.NewDecoder(r.Body)
		var chatReq openai.ChatRequest
		err := dec.Decode(&chatReq)
		if err != nil {
			l.Info("Error unmarshaling Body", err)
			http.Error(w, "error unmarshaling request", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			l.Warn("ResponseWriter does not support Flusher.")
			return
		}

		w.Header().Set("Content-Type", "application/json")

		prompt, err := prepareChatPrompt(chatReq.Messages)
		if err != nil {
			l.Info("Error preparing prompt", err)
			http.Error(w, "bad request (messages)", http.StatusBadRequest)
			return
		}

		stream := chatReq.Stream
		model := chatReq.Model

		req := Request{
			Prompt:      prompt,
			Stream:      stream,
			NPredict:    chatReq.MaxTokens,
			Temperature: chatReq.Temperature,
			TopP:        chatReq.TopP,
			CachePrompt: true,
			Stop:        stop,
			LogitBias:   [][2]float64{{523, -10.0}, {28789, -10.0}, {6647, -10.0}},
		}

		slot := queue.RequestSlot(session.SessionIdFromContext(ctx), req.Slot)
		defer queue.ReleaseSlot(slot)
		l = l.With(
			"slot", slot.ID,
			"userSlot", req.Slot,
			"endpointSlot", slot.endpointSlot.slot,
			"endpoint", slot.endpointSlot.endpoint,
		)
		l.Info("Got slot")

		llama := func(req Request, yield func([]byte) bool, stream bool) error {
			return handleLlamacpp(ctx, slot, req, yield, stream)
		}

		active, err := handleTools(
			w, llama, stream, llamacppRequestId, model, stop, l, chatReq, toolCalls, prepareChatPrompt)
		if err != nil {
			l.Error("Error in handleTools", err)
		}
		if active {
			// Return on active. Response has already been served by handleTools.
			return
		}

		if err := llama(
			req,
			func(line []byte) bool {
				content, finish_reason, err := extractFromLlamaLine(line)
				if err != nil {
					l.Error("Error parsing Llama.cpp response", err)
					http.Error(w, "Error parsing Llama.cpp response", http.StatusInternalServerError)
					return false
				}

				if err := writeChatCompletionResponse(
					w, stream, llamacppRequestId, model,
					openai.ChatCompletionMessage{Role: "assistant", Content: &content},
					finish_reason,
				); err != nil {
					l.Info("Error writing line", err)
					return false
				}
				flusher.Flush()
				return true
			},
			lineByLine,
		); err != nil {
			http.Error(w, "Error requesting response", http.StatusInternalServerError)
		}

		if stream {
			w.Write([]byte("\ndata: [DONE]"))
		}
		l.Info("Finished Response")
	})
}
