package llamacpp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/discovertomorrow/progai-middleware/pkg/handler"
	"github.com/discovertomorrow/progai-middleware/pkg/logging"
)

func handleLlamacpp(
	ctx context.Context,
	slot Slot,
	req Request,
	yield func([]byte) bool,
	lineByLine bool,
) error {
	l := logging.FromContext(ctx).With("function", "handler.handleLlamacpp")
	l.Debug("Call to llama.cpp competion backend", "pormpt", req.Prompt)

	req.Slot = slot.endpointSlot.slot
	if req.NPredict < 1 || req.NPredict > 2000 {
		req.NPredict = 2000
	}

	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(req); err != nil {
		l.Info("Error encoding request", err)
		return err
	}

	backendReq, err := http.NewRequestWithContext(
		logging.WithLogger(ctx, l),
		"POST",
		slot.endpointSlot.endpoint,
		bytes.NewBuffer(buf.Bytes()),
	)
	if err != nil {
		l.Error("Error creating request", err)
		return err
	}
	backendReq.Header.Set("Content-Type", "application/json")

	if err := handler.RequestBackend(
		backendReq,
		yield,
		lineByLine,
	); err != nil {
		l.Error("Error doing request", err)
		return err
	}

	return nil
}
