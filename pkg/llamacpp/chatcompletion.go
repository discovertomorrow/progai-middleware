package llamacpp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/discovertomorrow/progai-middleware/pkg/openai"
)

func writeChatCompletionResponse(
	w http.ResponseWriter,
	stream bool,
	llamacppRequestId string,
	model string,
	msg openai.ChatCompletionMessage,
	finish_reason *string,
) error {
	cr := createChatCompletionResponse(stream, llamacppRequestId, model, msg, finish_reason)
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(cr); err != nil {
		return err
	}
	if stream {
		if _, err := w.Write([]byte{10, 100, 97, 116, 97, 58, 32}); err != nil {
			return err
		}
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func createChatCompletionResponse(
	stream bool,
	llamacppRequestId string,
	model string,
	msg openai.ChatCompletionMessage,
	finish_reason *string,
) interface{} {
	var cr interface{}

	if stream {
		cr = openai.StreamChatResponse{
			Id:      llamacppRequestId,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []openai.StreamChatResponseChoice{
				{Delta: msg, FinishReason: finish_reason},
			},
		}
	} else {
		cr = openai.ChatResponse{
			Id:      llamacppRequestId,
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []openai.ChatResponseChoice{
				{
					Message:      msg,
					FinishReason: finish_reason,
				},
			},
		}
	}
	return cr
}

// extracts content and finish_reason
func extractFromLlamaLine(line []byte) (string, *string, error) {
	// remove "data: " prefix
	data := bytes.TrimPrefix(line, []byte{100, 97, 116, 97, 58, 32})
	if len(data) < 2 {
		return "", nil, nil
	}
	var r LlamaResponse
	err := json.Unmarshal(data, &r)
	if err != nil {
		return "", nil, err
	}
	var finish_reason *string
	if r.StoppedEos || r.StoppedWord {
		reason := "stop"
		finish_reason = &reason
	} else if r.StoppedLimit {
		reason := "length"
		finish_reason = &reason
	}
	return r.Content, finish_reason, nil
}
