package deploy

import (
	"net"
	"testing"

	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"google.golang.org/grpc"
)

func runDeployServer(t *testing.T) (*grpc.Server, string) {
	grpcServer := grpc.NewServer()

	serverSocketPath := "testdata/grpc.socket"
	listener, err := net.Listen("unix", serverSocketPath)

	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterDeployServiceServer(grpcServer, &server{})

	go grpcServer.Serve(listener)

	return grpcServer, serverSocketPath
}
