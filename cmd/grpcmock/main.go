// Command grpcmock is a sample target service that implements the taskd.v1.Runner
// contract. taskd (with protocol=grpc) will invoke Run for each task. Use it to
// smoke-test the gRPC executor end-to-end.
package main

import (
	"context"
	"fmt"
	"net"
	"os"

	taskdv1 "github.com/flametest/taskd/pkg/proto/taskdv1"
	"google.golang.org/grpc"
)

type runner struct {
	taskdv1.UnimplementedRunnerServer
}

func (r *runner) Run(ctx context.Context, req *taskdv1.RunRequest) (*taskdv1.RunResponse, error) {
	fmt.Fprintf(os.Stderr, "grpcmock: Run task_id=%s ref_id=%s params=%v\n", req.GetTaskId(), req.GetRefId(), req.GetParams().AsMap())
	return &taskdv1.RunResponse{}, nil
}

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:50052")
	if err != nil {
		panic(err)
	}
	gs := grpc.NewServer()
	taskdv1.RegisterRunnerServer(gs, &runner{})
	fmt.Fprintln(os.Stderr, "grpcmock listening on 127.0.0.1:50052")
	if err := gs.Serve(lis); err != nil {
		panic(err)
	}
}
