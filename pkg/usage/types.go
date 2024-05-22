package usage

import "context"

// Usage holds metrics related to the processing of an HTTP request.
type Usage struct {
	InputBytes          int
	OutputBytes         int
	InputToken          int
	InputTokenProcessed int
	OutputToken         int
	Images              int
}

// Creates and updates [Usage] metrics based on HTTP request data.
type UsageUpdater interface {
	// Extracts initial usage metrics from the provided request body.
	UsageFromInput(ctx context.Context, requestBody []byte) *Usage

	// Updates the usage metrics as the request is being processed. This method
	// can be called multiple times during a request's lifecycle.
	Update(ctx context.Context, usage *Usage, line string) error
}
