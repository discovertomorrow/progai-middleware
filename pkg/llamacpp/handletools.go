package llamacpp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/discovertomorrow/progai-middleware/pkg/openai"
)

func handleTools(
	w http.ResponseWriter,
	llama func(Request, func([]byte) bool, bool) error,
	stream bool,
	llamacppRequestId string,
	model string,
	stop []string,
	l *slog.Logger,
	chatReq openai.ChatRequest,
	toolCalls *ExpiringMap,
	prepareChatPrompt func([]openai.Message) (string, error),
) (bool, error) {
	if len(chatReq.Tools) == 0 || chatReq.ToolChoice == "none" {
		return false, nil
	}
	// prepare tool list for prompt, return if no tools
	lastTool, exists := getLastTool(chatReq.Messages, toolCalls)
	tools, exists := toolsToPrompt(chatReq.Tools, []string{lastTool})
	if !exists {
		return false, nil
	}
	l.Debug("Found Tools")
	if chatReq.ToolChoice != "required" {
		// check if a tool is helpful for the users request, return of not
		if !checkIfToolHelpful(llama, l, stop, prepareChatPrompt, chatReq.Messages, tools) {
			l.Debug("Finished Tools: Do NOT use Tool")
			return false, nil
		}
	}
	l.Debug("Get Tool Call")
	toolCall, err := generateToolCall(llama, l, stop, prepareChatPrompt, chatReq.Messages, tools)
	if err != nil {
		l.Error("Error generating tool call")
		return false, err
	}
	// the tool call is generated as a string of a python function call, parsing it
	name, arguments, err := parseFunction(toolCall)
	if err != nil {
		l.Error("Error parsing function")
		return false, err
	}
	var tool *openai.Tool
	for _, t := range chatReq.Tools {
		if t.Function.Name == name {
			tool = &t
			break
		}
	}
	if tool == nil {
		l.Error("Error: Tool not found", "name", name)
		return false, fmt.Errorf("Tool not found: %s", name)
	}

	// write http response: OpenAI API compatible tool response
	complMsg, finishReason, toolCallID := createToolChatcompletionMessage(tool.Function, arguments)
	writeChatCompletionResponse(w, stream, llamacppRequestId, model, complMsg, finishReason)
	// add toolCall to map
	toolCalls.Set(toolCallID, name)
	l.Debug("Finished Tools: Tool requested")
	return true, nil
}

func getLastTool(msgs []openai.Message, toolCalls *ExpiringMap) (string, bool) {
	last := msgs[len(msgs)-1]
	if last.Role == "tool" {
		return toolCalls.Get(*last.ToolCallID)
	}
	return "", false
}

func parseFunction(input string) (string, map[string]string, error) {
	// Find the function name and parameters
	re := regexp.MustCompile(`(\w+)\((.*)\)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 3 {
		return "", nil, fmt.Errorf("invalid input format")
	}

	funcName := matches[1]
	funcArguments := make(map[string]string)

	// Split the parameters by commas outside brackets
	re = regexp.MustCompile(`\w+=\[[^\[\]]*\]|\w+="[^"]*"|\w+=[^,]+`)
	params := re.FindAllString(matches[2], -1)

	for _, p := range params {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid parameter format")
		}

		paramName := strings.TrimSpace(parts[0])
		paramValue := strings.TrimSpace(parts[1])

		// Remove surrounding quotes from the value if present
		if strings.HasPrefix(paramValue, `"`) && strings.HasSuffix(paramValue, `"`) {
			paramValue = strings.Trim(paramValue, `"`)
		}

		funcArguments[paramName] = paramValue
	}

	return funcName, funcArguments, nil
}

func toolsToPrompt(tools []openai.Tool, ignoreTools []string) (string, bool) {
	var result []string
	for _, tool := range tools {
		if !contains(ignoreTools, tool.Function.Name) {
			result = append(result, toolToPrompt(tool))
		}
	}
	if len(result) == 0 {
		return "", false
	}
	return strings.Join(result, "\n"), true
}

func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func toolToPrompt(tool openai.Tool) string {
	var parameters []string
	var parDescs []string
	for parameter, value := range tool.Function.Parameters.Properties {
		parType := "str"
		if value.Type == "integer" {
			parType = "int"
		}
		parameters = append(parameters, fmt.Sprintf("%s: %s", parameter, parType))
		parDescs = append(parDescs, fmt.Sprintf("%s: %s %s", parameter, value.Description, strings.Join(value.Enum, ", ")))
	}
	return fmt.Sprintf(
		"%s(%s) # %s (%s)",
		tool.Function.Name,
		strings.Join(parameters, ", "),
		tool.Function.Description,
		strings.Join(parDescs, ", "))
}

func checkIfToolHelpful(
	llama func(Request, func([]byte) bool, bool) error,
	l *slog.Logger,
	stop []string,
	prepareChatPrompt func([]openai.Message) (string, error),
	msgs []openai.Message,
	tools string,
) bool {
	ml := len(msgs)
	ms := make([]openai.Message, ml+1)
	mu, exists := getLastUserMessage(msgs)
	if !exists {
		return false
	}
	if mu.Role != "user" {
		return false
	}
	copy(ms, msgs)
	ms[ml] = openai.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"Decide if it would be helpful to execute one of the "+
				"functions to answer the user question. Only consider the question "+
				"between \"<user-question></user-question>\". Decide now: "+
				"<functions>\n%s</functions> <user-question>%s</user-question> "+
				"Answer HELPFUL or NOT HELPFUL, nothing else. "+
				"If in doubt, choose NOT HELPFUL.",
			tools,
			mu.Content,
		),
	}
	l.Debug("Helpful desicion", "helpfulMessage", ms[ml-1])
	helpful := false
	yield := func(b []byte) bool {
		var r LlamaResponse
		err := json.Unmarshal(b, &r)
		if err != nil {
			l.Error("Error unmarshaling data", err)
			return false
		}
		helpful = strings.Trim(r.Content, " ") == "H"
		return true
	}
	prompt, err := prepareChatPrompt(ms)
	if err != nil {
		l.Debug("Error preparing function helpfulness prompt", err)
		return false
	}
	var temperature float32
	temperature = 0.01
	req := Request{
		Prompt:      prompt,
		Stream:      false,
		NPredict:    1,
		Temperature: &temperature,
		CachePrompt: true,
		Stop:        stop,
		LogitBias:   [][2]float64{{382, -0.3}},
	}
	if err := llama(req, yield, false); err != nil {
		l.Error("Error calling Backend")
		return false
	}
	return helpful
}

func getLastUserMessage(messages []openai.Message) (openai.Message, bool) {
	var mu *openai.Message
	for _, msg := range messages {
		if msg.Role == "user" {
			mu = &msg
		}
	}
	if mu == nil {
		return openai.Message{}, false
	}
	return *mu, true
}

func generateToolCall(
	llama func(Request, func([]byte) bool, bool) error,
	l *slog.Logger,
	stop []string,
	prepareChatPrompt func([]openai.Message) (string, error),
	msgs []openai.Message,
	tools string,
) (string, error) {
	var result string
	ml := len(msgs)
	ms := make([]openai.Message, ml+1)
	mu, exsist := getLastUserMessage(msgs)
	if !exsist {
		return "", fmt.Errorf("no user message found")
	}
	copy(ms, msgs)
	ms[ml] = openai.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"Use one of the following functions to answer the user question. "+
				"<functions>\n%s</functions> <user-question>%s</user-question> "+
				"Generate the function call. example: "+
				"CALL: height(building=\"Empire State Building\")",
			tools,
			mu.Content,
		),
	}
	prompt, err := prepareChatPrompt(ms)
	if err != nil {
		l.Debug("Error preparing function creation prompt", err)
		return "", err
	}
	var temperature float32
	temperature = 0.01
	if err := llama(
		Request{
			Prompt:      prompt,
			Stream:      false,
			NPredict:    90,
			Temperature: &temperature,
			CachePrompt: true,
			Stop:        stop,
		},
		func(b []byte) bool {
			var r LlamaResponse
			err := json.Unmarshal(b, &r)
			if err != nil {
				l.Error("Error unmarshaling data", err)
				return false
			}
			content, _ := strings.CutPrefix(strings.Trim(r.Content, " "), "CALL: ")
			result = strings.ReplaceAll(content, "\\_", "_")
			l.Debug("FUNCTION", "content", content)
			return true
		},
		false); err != nil {
		l.Error("Error calling Backend")
		return "", err
	}
	return result, nil
}

func createToolChatcompletionMessage(
	function openai.Function,
	toolArguments map[string]string,
) (openai.ChatCompletionMessage, *string, string) {
	var argStrings []string
	for k, v := range toolArguments {
		if p, ok := function.Parameters.Properties[k]; ok {
			if p.Type == "string" {
				argStrings = append(argStrings, fmt.Sprintf("\"%s\":\"%s\"", k, v))
			} else {
				argStrings = append(argStrings, fmt.Sprintf("\"%s\":%s", k, v))
			}
		}
	}
	toolCallId := strconv.FormatInt(time.Now().UnixNano(), 16)
	finish_reason := "tool_call"
	return openai.ChatCompletionMessage{
		Role: "assistant",
		ToolCalls: []openai.ToolCall{
			{
				Id:   toolCallId,
				Type: "function",
				Function: openai.ChatCompletionFunction{
					Name:      function.Name,
					Arguments: fmt.Sprintf("{%s}", strings.Join(argStrings, ",")),
				},
			},
		},
	}, &finish_reason, toolCallId
}
