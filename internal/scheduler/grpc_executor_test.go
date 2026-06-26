package scheduler

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/domain"
	taskdv1 "github.com/flametest/taskd/pkg/proto/taskdv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubRunner is a test Runner server that returns a configurable error.
type stubRunner struct {
	taskdv1.UnimplementedRunnerServer
	err error
}

func (s *stubRunner) Run(ctx context.Context, req *taskdv1.RunRequest) (*taskdv1.RunResponse, error) {
	return &taskdv1.RunResponse{}, s.err
}

// newGrpcServer starts a gRPC server serving srv on a random local port and
// returns its address plus a cleanup function.
func newGrpcServer(t *testing.T, srv taskdv1.RunnerServer) (addr string, cleanup func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	gs := grpc.NewServer()
	taskdv1.RegisterRunnerServer(gs, srv)
	go gs.Serve(lis)
	return lis.Addr().String(), gs.Stop
}

func newGrpcDom(addr string, proto enum.Protocol) *domain.Task {
	return &domain.Task{
		Id:       "t1",
		RefId:    "r1",
		Name:     "test",
		Protocol: proto,
		Address:  addr,
		Params:   map[string]interface{}{"k": "v"},
	}
}

func TestGrpcExecutor_Success(t *testing.T) {
	addr, cleanup := newGrpcServer(t, &stubRunner{})
	defer cleanup()

	exec := NewGrpcExecutor(2 * time.Second)
	if err := exec.Execute(context.Background(), newGrpcDom(addr, enum.ProtocolGRPC)); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestGrpcExecutor_RPCError(t *testing.T) {
	addr, cleanup := newGrpcServer(t, &stubRunner{err: status.Error(codes.Unavailable, "down")})
	defer cleanup()

	exec := NewGrpcExecutor(2 * time.Second)
	if err := exec.Execute(context.Background(), newGrpcDom(addr, enum.ProtocolGRPC)); err == nil {
		t.Fatal("expected error from Run, got nil")
	}
}

func TestGrpcExecutor_NonRetryableCode(t *testing.T) {
	addr, cleanup := newGrpcServer(t, &stubRunner{err: status.Error(codes.InvalidArgument, "bad arg")})
	defer cleanup()

	exec := NewGrpcExecutor(2 * time.Second)
	err := exec.Execute(context.Background(), newGrpcDom(addr, enum.ProtocolGRPC))
	if err == nil {
		t.Fatal("expected error from Run, got nil")
	}
	var nr *NonRetryableError
	if !errors.As(err, &nr) {
		t.Errorf("InvalidArgument should be a NonRetryableError, got %T: %v", err, err)
	}
}

func TestGrpcExecutor_RetryableCode(t *testing.T) {
	addr, cleanup := newGrpcServer(t, &stubRunner{err: status.Error(codes.Unavailable, "down")})
	defer cleanup()

	exec := NewGrpcExecutor(2 * time.Second)
	err := exec.Execute(context.Background(), newGrpcDom(addr, enum.ProtocolGRPC))
	if err == nil {
		t.Fatal("expected error from Run, got nil")
	}
	var nr *NonRetryableError
	if errors.As(err, &nr) {
		t.Errorf("Unavailable should be retryable, got NonRetryableError")
	}
}

func TestGrpcExecutor_UnsupportedProtocol(t *testing.T) {
	exec := NewGrpcExecutor(2 * time.Second)
	if err := exec.Execute(context.Background(), newGrpcDom("127.0.0.1:1", enum.ProtocolHTTP)); err == nil {
		t.Fatal("expected error for non-grpc protocol, got nil")
	}
}
