package openai

import (
	"encoding/json"
	"errors"
	"strings"
)

// OpenAI

type ChatRequest struct {
	Messages      []Message      `json:"messages"`
	Tools         []Tool         `json:"tools"`
	ToolChoice    string         `json:"tool_choice"`
	Model         string         `json:"model"`
	Stream        bool           `json:"stream"`
	MaxTokens     int            `json:"max_tokens"`
	Temperature   *float32       `json:"temperature"`
	TopP          *float32       `json:"top_p"`
	StreamOptions *StreamOptions `json:"stream_options"`
}

type StreamOptions map[string]interface{}

type Message struct {
	Role       string      `json:"role"`
	Content    Content     `json:"content"`
	Name       *string     `json:"name"`
	ToolCallID *string     `json:"tool_call_id"`
	ToolCalls  *[]ToolCall `json:"tool_calls"`
}

// Content is effectively a string in Go, but we give it a custom
// UnmarshalJSON method so we can handle both string input and
// array-of-blocks input in the incoming JSON.
type Content string

// UnmarshalJSON allows Content to handle two forms of JSON:
//  1. A raw string
//  2. An array of blocks, from which we take the `text` of the first block
//     whose `type` is `"text"`.
func (c *Content) UnmarshalJSON(data []byte) error {
	// --- First try: see if data is a simple JSON string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*c = Content(s)
		return nil
	}

	// --- Second try: parse data as an array of blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &blocks); err == nil {
		// Collect all text from blocks with `type="text"`
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" {
				parts = append(parts, b.Text)
			}
		}
		// Join with newline
		joined := strings.Join(parts, "\n")
		*c = Content(joined)
		return nil
	}

	return errors.New("content is neither a string nor an array of text blocks")
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type Parameters struct {
	Type       string              `json:"type"`
	Required   []string            `json:"required"`
	Properties map[string]Property `json:"properties"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum"`
}

type StreamChatResponseWithUsage struct {
	StreamChatResponse
	Usage *ChatResponseUsage `json:"usage"`
}

type StreamChatResponse struct {
	Id                string                     `json:"id"`
	Object            string                     `json:"object"`
	Created           int64                      `json:"created"`
	Model             string                     `json:"model"`
	SystemFingerprint string                     `json:"system_fingerprint"`
	Choices           []StreamChatResponseChoice `json:"choices"`
}

type StreamChatResponseChoice struct {
	Index        int         `json:"index"`
	Delta        interface{} `json:"delta"`
	Logprobs     *string     `json:"logprobs"`
	FinishReason *string     `json:"finish_reason"`
}

type EmptyDelta struct{}

type ChatResponse struct {
	Id                string               `json:"id"`
	Object            string               `json:"object"`
	Created           int64                `json:"created"`
	Model             string               `json:"model"`
	SystemFingerprint string               `json:"system_fingerprint"`
	Choices           []ChatResponseChoice `json:"choices"`
	Usage             ChatResponseUsage    `json:"usage,omitempty"`
}

type ChatResponseChoice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	Logprobs     *string               `json:"logprobs"`
	FinishReason *string               `json:"finish_reason"`
}

type ChatResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionMessage struct {
	Role      string     `json:"role"`
	Content   *string    `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type ToolCall struct {
	Id       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function ChatCompletionFunction `json:"function"`
}

type ChatCompletionFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
