package scheduler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/domain"
)

func newHTTPDom(addr string, proto enum.Protocol) *domain.Task {
	return &domain.Task{
		Id:       "t1",
		Name:     "test",
		Protocol: proto,
		Address:  addr,
		Params:   map[string]interface{}{"k": "v"},
	}
}

func TestHTTPExecutor_Success(t *testing.T) {
	var gotMethod, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	if _, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %s, want application/json", gotCT)
	}
}

func TestHTTPExecutor_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	if _, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

func TestHTTPExecutor_4xxNonRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	_, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP))
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	var nr *NonRetryableError
	if !errors.As(err, &nr) {
		t.Errorf("404 should be a NonRetryableError, got %T: %v", err, err)
	}
}

func TestHTTPExecutor_5xxRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	_, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP))
	if err == nil {
		t.Fatal("expected error on 502, got nil")
	}
	var nr *NonRetryableError
	if errors.As(err, &nr) {
		t.Errorf("502 should be retryable, got NonRetryableError")
	}
}

func TestHTTPExecutor_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(20 * time.Millisecond) // short timeout
	if _, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestHTTPExecutor_ConnectionRefused(t *testing.T) {
	// Close the server immediately so the address refuses connections.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	if _, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestHTTPExecutor_UnsupportedProtocol(t *testing.T) {
	exec := NewHTTPExecutor(2 * time.Second)
	if _, err := exec.Execute(context.Background(), newHTTPDom("grpc://x", enum.ProtocolGRPC)); err == nil {
		t.Fatal("expected error for grpc protocol, got nil")
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		addr  string
		proto enum.Protocol
		want  string
	}{
		{"127.0.0.1:8080/actuator/health", enum.ProtocolHTTP, "http://127.0.0.1:8080/actuator/health"},
		{"127.0.0.1:8080", enum.ProtocolHTTPS, "https://127.0.0.1:8080"},
		{"http://example.com/x", enum.ProtocolHTTP, "http://example.com/x"},
		{"https://example.com/x", enum.ProtocolHTTPS, "https://example.com/x"},
	}
	for _, c := range cases {
		if got := normalizeURL(c.addr, c.proto); got != c.want {
			t.Errorf("normalizeURL(%q, %s) = %q, want %q", c.addr, c.proto, got, c.want)
		}
	}
}

// TestHTTPExecutor_BareAddress verifies a bare address (no scheme) reaches the
// upstream instead of failing at url.Parse. Regression test for the
// "first path segment in URL cannot contain colon" bug.
func TestHTTPExecutor_BareAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/actuator/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// Strip "http://" to simulate a bare address like "127.0.0.1:port/actuator/health".
	bare := strings.TrimPrefix(srv.URL, "http://") + "/actuator/health"
	exec := NewHTTPExecutor(2 * time.Second)
	if _, err := exec.Execute(context.Background(), newHTTPDom(bare, enum.ProtocolHTTP)); err != nil {
		t.Fatalf("Execute with bare address: %v", err)
	}
}

func TestHTTPExecutor_CapturesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	resp, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.Status != "200" {
		t.Errorf("Status = %q, want 200", resp.Status)
	}
	if resp.Body != `{"ok":true}` {
		t.Errorf("Body = %q, want {\"ok\":true}", resp.Body)
	}
}

// TestHTTPExecutor_CapturesResponseOnError verifies the body is captured even on
// 4xx, so the audit log shows why the upstream rejected the request.
func TestHTTPExecutor_CapturesResponseOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad input"))
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	resp, err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP))
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if resp == nil {
		t.Fatal("response is nil even on 4xx (body should be captured)")
	}
	if resp.Status != "400" {
		t.Errorf("Status = %q, want 400", resp.Status)
	}
	if resp.Body != "bad input" {
		t.Errorf("Body = %q, want 'bad input'", resp.Body)
	}
}
