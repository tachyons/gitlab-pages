package deploy

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	serverSocketPath = "testdata/grpc.socket"
	testRootDir      = "testdata/root"
)

func TestDeleteSite(t *testing.T) {
	sitePath := "foo/bar"
	testSiteDir := path.Join(testRootDir, sitePath)
	require.NoError(t, os.RemoveAll(testSiteDir))
	require.NoError(t, os.MkdirAll(testSiteDir, 0755))

	require.NoError(t, ioutil.WriteFile(path.Join(testSiteDir, "hello"), []byte("world"), 0644))

	s := runDeployServer(t)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, conn := newDeployClient(t)
	defer conn.Close()

	_, err := client.DeleteSite(ctx, &pb.DeleteSiteRequest{Path: sitePath})
	require.NoError(t, err)

	_, err = os.Stat(testSiteDir)
	require.True(t, os.IsNotExist(err), "directory should have been removed")

	_, err = os.Stat(path.Dir(testSiteDir))
	require.NoError(t, err, "parent directory should still exist")
}

func TestDeleteSiteFail(t *testing.T) {
	s := runDeployServer(t)
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, conn := newDeployClient(t)
	defer conn.Close()

	testCases := []struct {
		desc string
		path string
		code codes.Code
	}{
		{desc: "empty path", path: "", code: codes.InvalidArgument},
		{desc: "traversal beginning", path: "../foo", code: codes.InvalidArgument},
		{desc: "traversal middle", path: "bar/../foo", code: codes.InvalidArgument},
		{desc: "traversal end", path: "foo/bar/..", code: codes.InvalidArgument},
		{desc: "path starting with period", path: ".foo/bar", code: codes.InvalidArgument},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			req := &pb.DeleteSiteRequest{Path: tc.path}

			_, err := client.DeleteSite(ctx, req)
			st, ok := status.FromError(err)
			require.True(t, ok, "error has a grpc status")
			require.Equal(t, tc.code, st.Code(), "unexpected grpc code")
		})
	}
}

func runDeployServer(t *testing.T) *grpc.Server {
	grpcServer := grpc.NewServer()

	listener, err := net.Listen("unix", serverSocketPath)

	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterDeployServiceServer(grpcServer, &server{rootDir: testRootDir})

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
