package deploy

import (
	"context"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	serverSocketPath = "../grpc.socket"
	testRootDir      = "testdata/root"
)

var (
	cdRootOnce sync.Once
)

// cdRoot changes the working directory of the test executable. We are
// forced to assume that the pages root directory is the current working
// directory. When running pages with chroot+bind mount, os.Getwd()
// resolves to a garbage "(undefined)" vaule. So in turn, the tests for
// this package must execute with the pages root as the working
// directory.
func cdRoot(t *testing.T) {
	cdRootOnce.Do(func() {
		require.NoError(t, os.Chdir(testRootDir))
	})
}

func TestDeleteSite(t *testing.T) {
	cdRoot(t)

	sitePath, testSiteDir := setupTestSite(t)
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

func setupTestSite(t *testing.T) (sitePath string, testSiteDir string) {
	sitePath = "foo/bar"
	testSiteDir, err := filepath.Abs(sitePath)
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(testSiteDir))
	require.NoError(t, os.MkdirAll(testSiteDir, 0755))

	return sitePath, testSiteDir
}

func TestDeleteSiteFail(t *testing.T) {
	cdRoot(t)

	sitePath, testSiteDir := setupTestSite(t)
	require.NoError(t, ioutil.WriteFile(path.Join(testSiteDir, "hello"), []byte("world"), 0644))

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
		{desc: "path starting with slash", path: "/foo/bar", code: codes.InvalidArgument},
		{desc: "directory does not exist", path: "does/not/exist", code: codes.FailedPrecondition},
		{desc: "path is a file not a directory", path: path.Join(sitePath, "hello"), code: codes.FailedPrecondition},
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

	pb.RegisterDeployServiceServer(grpcServer, NewServer())

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
