package handler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/discovertomorrow/progai-middleware/pkg/logging"
)

func RequestBackend(
	req *http.Request,
	yield func([]byte) bool,
	lineByLine bool,
) error {
	l := logging.FromContext(req.Context()).With("function", "handler.responseToChannel")

	client := &http.Client{
		Timeout: 300 * time.Second,
	}
	l.Info("Sending request to backend")
	resp, err := client.Do(req)
	if err != nil {
		l.Error("Error sending request", err)
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if lineByLine {
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					if len(line) > 0 {
						if !yield(line) {
							return fmt.Errorf("writing response")
						}
					}
					break // End of file
				}
				l.Error("Error reading response", err)
				return fmt.Errorf("reading response: %w", err)
			}
			if !yield(line) {
				return fmt.Errorf("writing response")
			}
			if err != nil && err == io.EOF {
				break // End of file
			}
		}
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			l.Error("Error reading body", err)
			return fmt.Errorf("reading body: %w", err)
		}
		if !yield(body) {
			return fmt.Errorf("writing body")
		}
	}
	return nil
}
