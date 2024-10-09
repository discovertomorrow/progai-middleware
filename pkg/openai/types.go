package openai

// OpenAI

type ChatRequest struct {
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools"`
	ToolChoice  string    `json:"tool_choice"`
	Model       string    `json:"model"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature *float32  `json:"temperature"`
	TopP        *float32  `json:"top_p"`
}

type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	Name       *string     `json:"name"`
	ToolCallID *string     `json:"tool_call_id"`
	ToolCalls  *[]ToolCall `json:"tool_calls"`
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

type StreamChatResponse struct {
	Id                string                     `json:"id"`
	Object            string                     `json:"object"`
	Created           int64                      `json:"created"`
	Model             string                     `json:"model"`
	SystemFingerprint string                     `json:"system_fingerprint"`
	Choices           []StreamChatResponseChoice `json:"choices"`
}

type StreamChatResponseChoice struct {
	Index        int                   `json:"index"`
	Delta        ChatCompletionMessage `json:"delta"`
	Logprobs     *string               `json:"logprobs"`
	FinishReason *string               `json:"finish_reason"`
}

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
