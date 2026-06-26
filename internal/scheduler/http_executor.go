package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/vita/verrors"
)

// HTTPExecutor runs a task by POSTing its Params as a JSON body to Address. Only
// http/https are supported this round; any other protocol returns an error (the
// task retries toward dead). A non-nil error (network, timeout, or HTTP >=400)
// triggers the retry path; a nil error triggers MarkSucceeded.
type HTTPExecutor struct {
	client *http.Client
}

// NewHTTPExecutor builds an HTTPExecutor with the given per-call timeout.
func NewHTTPExecutor(timeout time.Duration) *HTTPExecutor {
	return &HTTPExecutor{client: &http.Client{Timeout: timeout}}
}

func (e *HTTPExecutor) Execute(ctx context.Context, task *domain.Task) error {
	if task.Protocol != enum.ProtocolHTTP && task.Protocol != enum.ProtocolHTTPS {
		return verrors.NotImplementedError(fmt.Sprintf("protocol %s not supported", task.Protocol))
	}
	body, err := json.Marshal(task.Params)
	if err != nil {
		return verrors.BadRequestError(fmt.Sprintf("marshal params: %v", err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, task.Address, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return err // connect / timeout / DNS -> retry
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("upstream returned %d", resp.StatusCode)
		if resp.StatusCode < 500 {
			// 4xx: client error, retrying won't help -> non-retryable (dead).
			return NewNonRetryableError(verrors.BadRequestError(msg))
		}
		// 5xx: server error -> retryable.
		return verrors.InternalServerError(msg)
	}
	return nil
}
