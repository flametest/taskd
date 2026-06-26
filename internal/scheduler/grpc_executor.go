package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/domain"
	taskdv1 "github.com/flametest/taskd/pkg/proto/taskdv1"
	"github.com/flametest/vita/verrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// GrpcExecutor runs a task by invoking the Runner/Run RPC on the target service
// at Address. Only protocol grpc is supported; any RPC error is returned so the
// scheduler retries. A fresh connection is used per call (simple and stateless;
// a connection pool can be added later if needed).
type GrpcExecutor struct {
	timeout time.Duration
}

func NewGrpcExecutor(timeout time.Duration) *GrpcExecutor {
	return &GrpcExecutor{timeout: timeout}
}

func (e *GrpcExecutor) Execute(ctx context.Context, task *domain.Task) error {
	if task.Protocol != enum.ProtocolGRPC {
		return verrors.NotImplementedError(fmt.Sprintf("protocol %s not supported by GrpcExecutor", task.Protocol))
	}
	conn, err := grpc.NewClient(task.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := taskdv1.NewRunnerClient(conn)
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	params, err := structpb.NewStruct(task.Params)
	if err != nil {
		return verrors.BadRequestError(fmt.Sprintf("convert params: %v", err))
	}
	_, err = client.Run(ctx, &taskdv1.RunRequest{TaskId: task.Id, RefId: task.RefId, Params: params})
	if err == nil {
		return nil
	}
	if isRetryableGrpcStatus(status.Convert(err)) {
		return err
	}
	return NewNonRetryableError(err)
}

// isRetryableGrpcStatus reports whether a gRPC status is worth retrying. Codes
// that signal a client/logic error (InvalidArgument, NotFound, ...) are not
// retryable; transient/server errors (Unavailable, DeadlineExceeded, ...) are.
func isRetryableGrpcStatus(s *status.Status) bool {
	switch s.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted,
		codes.Aborted, codes.Internal, codes.Unknown, codes.Canceled, codes.DataLoss:
		return true
	default:
		return false
	}
}

// CompositeExecutor dispatches Execute to a per-protocol executor (HTTP or gRPC).
type CompositeExecutor struct {
	httpExec Executor
	grpcExec Executor
}

// NewCompositeExecutor builds a CompositeExecutor over the given HTTP and gRPC
// executors.
func NewCompositeExecutor(httpExec, grpcExec Executor) *CompositeExecutor {
	return &CompositeExecutor{httpExec: httpExec, grpcExec: grpcExec}
}

func (c *CompositeExecutor) Execute(ctx context.Context, task *domain.Task) error {
	switch task.Protocol {
	case enum.ProtocolHTTP, enum.ProtocolHTTPS:
		return c.httpExec.Execute(ctx, task)
	case enum.ProtocolGRPC:
		return c.grpcExec.Execute(ctx, task)
	default:
		return verrors.NotImplementedError(fmt.Sprintf("protocol %s not supported", task.Protocol))
	}
}
