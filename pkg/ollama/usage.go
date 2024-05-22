package ollama

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/discovertomorrow/progai-middleware/pkg/logging"
	"github.com/discovertomorrow/progai-middleware/pkg/usage"
)

const ollamaPattern string = `"prompt_eval_count":(\d+),"prompt_eval_duration":(\d+),` +
	`"eval_count":(\d+),"eval_duration":(\d+)}`

type OllamaUsageUpdater struct {
}

func NewOllamaUsageUpdater() *OllamaUsageUpdater {
	return &OllamaUsageUpdater{}
}

func (u *OllamaUsageUpdater) UsageFromInput(
	ctx context.Context,
	requestBody []byte,
) *usage.Usage {
	return &usage.Usage{
		InputBytes: len(requestBody),
	}
}

func (u *OllamaUsageUpdater) Update(ctx context.Context, usage *usage.Usage, line string) error {
	l := logging.FromContext(ctx).With("usage", usage, "line", line)
	l.Debug("Update")
	usage.OutputBytes += len(line)

	re, err := regexp.Compile(ollamaPattern)
	if err != nil {
		l.Debug("regex compile error", "error", err)
		return err
	}
	matches := re.FindStringSubmatch(line)
	if matches == nil {
		l.Debug("no match")
		return fmt.Errorf("no matches found")
	}
	tokensEvaluated, err := strconv.Atoi(matches[1])
	if err != nil {
		l.Warn("tokensEvaluated conversion failed", "error", err)
		return err
	}
	tokensPredicted, err := strconv.Atoi(matches[3])
	if err != nil {
		l.Warn("tokensPredicted conversion failed", "error", err)
		return err
	}
	l.Debug("Updated", "tokensEvaluated", tokensEvaluated, "tokensPredicted", tokensPredicted)
	usage.InputToken += tokensEvaluated
	usage.OutputToken += tokensPredicted
	return nil
}
