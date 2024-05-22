package llamacpp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/discovertomorrow/progai-middleware/pkg/logging"
	"github.com/discovertomorrow/progai-middleware/pkg/usage"
)

type LlamacppUsageUpdater struct {
}

func NewLlamacppUsageUpdater() *LlamacppUsageUpdater {
	return &LlamacppUsageUpdater{}
}

type LlamaRequest struct {
	Prompt string `json:"prompt"`
}

type LlamaResponse struct {
	Content         string               `json:"content"`
	Stop            bool                 `json:"stop"`
	StoppedEos      bool                 `json:"stopped_eos"`
	StoppedLimit    bool                 `json:"stopped_limit"`
	StoppedWord     bool                 `json:"stopped_word"`
	TokensCached    int                  `json:"tokens_cached"`
	TokensEvaluated int                  `json:"tokens_evaluated"`
	TokensPredicted int                  `json:"tokens_predicted"`
	Timings         LlamaResponseTimings `json:"timings"`
}

type LlamaResponseTimings struct {
	PredictedN int `json:"predicted_n"`
	PromptN    int `json:"prompt_n"`
}

func (u *LlamacppUsageUpdater) UsageFromInput(
	ctx context.Context,
	requestBody []byte,
) *usage.Usage {
	var r LlamaRequest
	err := json.Unmarshal(requestBody, &r)
	if err != nil {
		return &usage.Usage{}
	}
	return &usage.Usage{
		InputBytes: len(r.Prompt),
	}
}

func (u *LlamacppUsageUpdater) Update(
	ctx context.Context,
	usage *usage.Usage,
	line string,
) error {
	l := logging.FromContext(ctx).With("usage", usage, "line", line)
	l.Debug("Update")
	if line == "" {
		return nil
	}

	jsonLine := strings.TrimPrefix(line, "data: ")

	var r LlamaResponse
	err := json.Unmarshal([]byte(jsonLine), &r)
	if err != nil {
		return fmt.Errorf("unmarshaling line: %w", err)
	}

	usage.OutputBytes += len(r.Content)

	if len(r.Content) > 0 {
		// in streaming, if we are not at stop but abort, we will be able to report
		// generated tokens.
		usage.OutputToken++
	}

	if r.Stop {
		// llamacpp outputs usage statstics if stop is true.
		usage.InputToken = r.TokensEvaluated
		usage.InputTokenProcessed = r.Timings.PromptN
		usage.OutputToken = r.Timings.PredictedN
	}
	return nil
}
