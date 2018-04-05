package deploy

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"google.golang.org/grpc"
)

var (
	serverSocketPath = "testdata/grpc.socket"
)

func TestDeleteSite(t *testing.T) {
	s := runDeployServer(t)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, conn := newDeployClient(t)
	defer conn.Close()

	req := &pb.DeleteSiteRequest{}

	_, err := client.DeleteSite(ctx, req)
	require.NoError(t, err)
}

func runDeployServer(t *testing.T) *grpc.Server {
	grpcServer := grpc.NewServer()

	listener, err := net.Listen("unix", serverSocketPath)

	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterDeployServiceServer(grpcServer, &server{})

	go grpcServer.Serve(listener)

	return grpcServer
}

func newDeployClient(t *testing.T) (pb.DeployServiceClient, *grpc.ClientConn) {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return pb.NewDeployServiceClient(conn), conn
}
