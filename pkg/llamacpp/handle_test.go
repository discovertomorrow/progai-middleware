package llamacpp

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/discovertomorrow/progai-middleware/pkg/handler"
	"github.com/stretchr/testify/mock"
)

// Mock function type
type MockHandleFunc struct {
	mock.Mock
}

func (m *MockHandleFunc) Handle(
	ctx context.Context,
	slot Slot,
	req Request,
	writeLine func(line []byte) bool,
	lineByLine bool,
) error {
	args := m.Called(ctx, slot, req, writeLine, lineByLine)
	return args.Error(0)
}

// Setup function to initialize common objects
func setup() (*MockHandleFunc, []handler.Endpoint) {
	mockHandle := new(MockHandleFunc)
	endpoints := []handler.Endpoint{
		{Endpoint: "http://localhost:8080", Parallel: 1},
	}

	return mockHandle, endpoints
}

func TestNewLlamacppHandler(t *testing.T) {
	mockHandle, endpoints := setup()

	handler := newLlamacppHandlerInternal(
		true,
		mockHandle.Handle,
		NewQueue(endpoints),
	)

	// Test case where prompt is provided
	t.Run("WithPrompt", func(t *testing.T) {
		reqBody := `{"prompt": "Hi"}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		mockHandle.On(
			"Handle",
			mock.Anything,
			mock.MatchedBy(func(slot Slot) bool {
				return slot.endpointSlot.endpoint == "http://localhost:8080"
			}),
			mock.MatchedBy(func(req Request) bool {
				return req.Prompt == "Hi"
			}),
			mock.Anything,
			mock.Anything,
		).Return(nil)

		handler.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status OK; got %v", resp.Status)
		}

		mockHandle.AssertExpectations(t)
	})

	// Test case where prompt is missing
	t.Run("WithoutPrompt", func(t *testing.T) {
		reqBody := `{}` // Missing prompt
		req := httptest.NewRequest("POST", "/", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		mockHandle.On(
			"Handle",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(nil)

		handler.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400; got %v", resp.Status)
		}
	})
}

func TestNewLlamacppChatHandler(t *testing.T) {
	chatTemplate := `{{ range . }}
{{- if eq .Role "system" }} [INST] <system> {{ .Content }} </system> [/INST]
{{- else if eq .Role "user" }} [INST] {{ .Content }} [/INST]
{{- else if eq .Role "tool" }} [INST] <toolresult> {{ .Content }} </toolresult> [/INST]
{{- else }} {{ .Content }}
{{- if .ToolCalls }}{{ range .ToolCalls }}<toolcall> {{ .Function.Name }} with arguments {{ .Function.Arguments }} </toolcall> {{ end }}{{ end -}}
</s>{{ end }}
{{- end }}`
	stop := []string{"</s>"}
	mockHandle, endpoints := setup()

	handler := newLlamacppChatHandlerInternal(
		slog.Default(),
		true,
		chatTemplate,
		stop,
		mockHandle.Handle,
		NewQueue(endpoints),
	)

	// Test case where prompt is provided
	t.Run("WithPrompt", func(t *testing.T) {
		reqBody := `{
  "model": "gpt-4o",
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful assistant."
    },
    {
      "role": "user",
      "content": "Hello!"
    }
  ]
}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		mockHandle.On(
			"Handle",
			mock.Anything,
			mock.MatchedBy(func(slot Slot) bool {
				return slot.endpointSlot.endpoint == "http://localhost:8080"
			}),
			mock.MatchedBy(func(req Request) bool {
				return req.Prompt == " [INST] <system> You are a helpful assistant. </system> [/INST] [INST] Hello! [/INST]"
			}),
			mock.Anything,
			mock.Anything,
		).Return(nil)

		handler.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status OK; got %v", resp.Status)
		}

		mockHandle.AssertExpectations(t)
	})

	// Test case where prompt is missing
	t.Run("WithZeroMessages", func(t *testing.T) {
		reqBody := `{
  "model": "gpt-4o",
  "messages": []
}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		mockHandle.On(
			"Handle",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(nil)

		handler.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400; got %v", resp.Status)
		}
	})
	t.Run("WithoutMessages", func(t *testing.T) {
		reqBody := `{
  "model": "gpt-4o"
}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(reqBody))
		w := httptest.NewRecorder()

		mockHandle.On(
			"Handle",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(nil)

		handler.ServeHTTP(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400; got %v", resp.Status)
		}
	})
}
