package usage

import "context"

// Provides a basic implementation of [UsageUpdater]
type DefaultUsageUpdater struct{}

func NewDefaultUsageUpdater() *DefaultUsageUpdater {
	return &DefaultUsageUpdater{}
}

// Implements [UsageUpdater.UsageFromInput].
func (u *DefaultUsageUpdater) UsageFromInput(
	ctx context.Context,
	requestBody []byte,
) *Usage {
	return &Usage{
		InputBytes: len(requestBody),
	}
}

// Implements [UsageUpdater.Update].
func (u *DefaultUsageUpdater) Update(ctx context.Context, usage *Usage, line string) error {
	usage.OutputBytes += len(line)
	return nil
}
