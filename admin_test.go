package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

var (
	// Use ../../ because the pages binary interprets the path in ./shared/pages
	adminSecretArgs = []string{"-admin-secret-path", "../../testdata/.admin-secret"}
	adminToken      = "super-secret\n"
)

func TestAdminHealthCheckUnix(t *testing.T) {
	socketPath := "admin.socket"
	// Use "../../" because the pages executable cd's into shared/pages
	adminArgs := append(adminSecretArgs, "-admin-unix-listener", "../../"+socketPath)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, listeners, "", adminArgs...)
	defer teardown()

	waitHTTP2RoundTripUnix(t, socketPath)

	testCases := []struct {
		desc    string
		dialOpt grpc.DialOption
		code    codes.Code
	}{
		{
			desc: "no auth provided",
			code: codes.Unauthenticated,
		},
		{
			desc:    "wrong auth provided",
			dialOpt: grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials("wrong token")),
			code:    codes.PermissionDenied,
		},
		{
			desc:    "correct auth provided",
			dialOpt: grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials(adminToken)),
			code:    codes.OK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			connOpts := []grpc.DialOption{
				grpc.WithInsecure(),
				grpcUnixDialOpt(),
			}
			if tc.dialOpt != nil {
				connOpts = append(connOpts, tc.dialOpt)
			}

			conn, err := grpc.Dial(socketPath, connOpts...)
			require.NoError(t, err, "dial")
			defer conn.Close()

			err = healthCheck(conn)
			require.Equal(t, tc.code, status.Code(err), "wrong grpc code: %v", err)
		})
	}
}

func TestAdminHealthCheckHTTPS(t *testing.T) {
	key, cert := CreateHTTPSFixtureFiles(t)
	creds, err := credentials.NewClientTLSFromFile(cert, "")
	require.NoError(t, err, "grpc client credentials")

	adminAddr := newAddr()
	adminArgs := []string{"-admin-https-listener", adminAddr, "-admin-https-key", key, "-admin-https-cert", cert}
	adminArgs = append(adminArgs, adminSecretArgs...)

	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, listeners, "", adminArgs...)
	defer teardown()

	waitHTTP2RoundTrip(t, adminAddr)

	testCases := []struct {
		desc    string
		dialOpt grpc.DialOption
		code    codes.Code
	}{
		{
			desc: "no auth provided",
			code: codes.Unauthenticated,
		},
		{
			desc:    "wrong auth provided",
			dialOpt: grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials("wrong token")),
			code:    codes.PermissionDenied,
		},
		{
			desc:    "correct auth provided",
			dialOpt: grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials(adminToken)),
			code:    codes.OK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			connOpts := []grpc.DialOption{
				grpc.WithTransportCredentials(creds),
			}
			if tc.dialOpt != nil {
				connOpts = append(connOpts, tc.dialOpt)
			}
			conn, err := grpc.Dial(adminAddr, connOpts...)
			require.NoError(t, err, "dial")
			defer conn.Close()

			err = healthCheck(conn)
			require.Equal(t, tc.code, status.Code(err), "wrong grpc code: %v", err)
		})
	}
}

func newAddr() string {
	s := httptest.NewServer(http.NotFoundHandler())
	s.Close()
	return s.Listener.Addr().String()
}

func waitHTTP2RoundTrip(t *testing.T, addr string) {
	transport := &http2.Transport{
		TLSClientConfig: &tls.Config{RootCAs: TestCertPool},
	}

	req, err := http.NewRequest("get", "https://"+addr, nil)
	require.NoError(t, err)

	for start := time.Now(); time.Since(start) < 5*time.Second; time.Sleep(100 * time.Millisecond) {
		var response *http.Response
		response, err = transport.RoundTrip(req)
		if err == nil {
			response.Body.Close()
			return
		}
	}

	t.Fatal(err)
}

func grpcUnixDialOpt() grpc.DialOption {
	return grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	})
}

func waitHTTP2RoundTripUnix(t *testing.T, socketPath string) {
	var err error

	for start := time.Now(); time.Since(start) < 5*time.Second; time.Sleep(100 * time.Millisecond) {
		err = roundtripHTTP2Unix(socketPath)
		if err == nil {
			return
		}
	}

	t.Fatal(err)
}

func roundtripHTTP2Unix(socketPath string) error {
	transport := &http2.Transport{
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	req, err := http.NewRequest("get", "https://localhost/", nil)
	if err != nil {
		return err
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func healthCheck(conn *grpc.ClientConn) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := healthpb.NewHealthClient(conn)
	_, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
	return err
}
