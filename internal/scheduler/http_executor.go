package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/vita/verrors"
)

// maxResponseBodyLen caps the response body stored in task_record.response so a
// large upstream reply cannot inflate the audit row.
const maxResponseBodyLen = 8 << 10 // 8 KiB

// HTTPExecutor runs a task by POSTing its Params as a JSON body to Address. Only
// http/https are supported; any other protocol returns an error (the task
// retries toward dead). A non-nil error (network, timeout, or HTTP >=400)
// triggers the retry path; a nil error triggers MarkSucceeded.
type HTTPExecutor struct {
	client *http.Client
}

// NewHTTPExecutor builds an HTTPExecutor with the given per-call timeout.
func NewHTTPExecutor(timeout time.Duration) *HTTPExecutor {
	return &HTTPExecutor{client: &http.Client{Timeout: timeout}}
}

func (e *HTTPExecutor) Execute(ctx context.Context, task *domain.Task) (*ExecutionResponse, error) {
	if task.Protocol != enum.ProtocolHTTP && task.Protocol != enum.ProtocolHTTPS {
		return nil, verrors.NotImplementedError(fmt.Sprintf("protocol %s not supported", task.Protocol))
	}
	body, err := json.Marshal(task.Params)
	if err != nil {
		return nil, verrors.BadRequestError(fmt.Sprintf("marshal params: %v", err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, normalizeURL(task.Address, task.Protocol), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err // connect / timeout / DNS -> retry, no response captured
	}
	defer resp.Body.Close()

	// Read up to maxResponseBodyLen+1 bytes so truncation is detectable.
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyLen+1))
	execResp := &ExecutionResponse{
		Status: strconv.Itoa(resp.StatusCode),
		Body:   truncateErr(string(raw), maxResponseBodyLen),
	}
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("upstream returned %d", resp.StatusCode)
		if resp.StatusCode < 500 {
			// 4xx: client error, retrying won't help -> non-retryable (dead).
			return execResp, NewNonRetryableError(verrors.BadRequestError(msg))
		}
		// 5xx: server error -> retryable.
		return execResp, verrors.InternalServerError(msg)
	}
	return execResp, nil
}

// normalizeURL ensures addr carries a scheme. A bare host[:port]/path such as
// "127.0.0.1:8080/actuator/health" is otherwise parsed as a path by url.Parse,
// failing with "first path segment in URL cannot contain colon". The scheme is
// derived from the task protocol when addr has no "://".
func normalizeURL(addr string, protocol enum.Protocol) string {
	if strings.Contains(addr, "://") {
		return addr
	}
	if protocol == enum.ProtocolHTTPS {
		return "https://" + addr
	}
	return "http://" + addr
}
