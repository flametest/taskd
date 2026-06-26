package scheduler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
	if err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err != nil {
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
	if err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

func TestHTTPExecutor_4xxNonRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP))
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
	err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP))
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
	if err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestHTTPExecutor_ConnectionRefused(t *testing.T) {
	// Close the server immediately so the address refuses connections.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	exec := NewHTTPExecutor(2 * time.Second)
	if err := exec.Execute(context.Background(), newHTTPDom(srv.URL, enum.ProtocolHTTP)); err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestHTTPExecutor_UnsupportedProtocol(t *testing.T) {
	exec := NewHTTPExecutor(2 * time.Second)
	if err := exec.Execute(context.Background(), newHTTPDom("grpc://x", enum.ProtocolGRPC)); err == nil {
		t.Fatal("expected error for grpc protocol, got nil")
	}
}
