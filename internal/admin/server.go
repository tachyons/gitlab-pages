package admin

import (
	"crypto/tls"
	"os"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	log "github.com/sirupsen/logrus"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"gitlab.com/gitlab-org/gitlab-pages/internal/service/deploy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

var logrusEntry *log.Entry

func init() {
	logger := log.StandardLogger()

	logrusEntry = log.NewEntry(logger)
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)
}

// NewServer creates a new unencrypted gRPC server for the gitlab-pages admin API.
func NewServer(secret string) *grpc.Server {
	grpcServer := grpc.NewServer(serverOpts(secret)...)
	registerServices(grpcServer)
	return grpcServer
}

// NewTLSServer creates a new gRPC server with encryption for the gitlab-pages admin API.
func NewTLSServer(secret string, cert *tls.Certificate) *grpc.Server {
	grpcServer := grpc.NewServer(append(
		serverOpts(secret),
		grpc.Creds(credentials.NewServerTLSFromCert(cert)),
	)...)
	registerServices(grpcServer)
	return grpcServer
}

func serverOpts(secret string) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_prometheus.StreamServerInterceptor,
			grpc_logrus.StreamServerInterceptor(logrusEntry),
			grpc_auth.StreamServerInterceptor(authFunc(secret)),
			grpc_recovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_logrus.UnaryServerInterceptor(logrusEntry),
			grpc_auth.UnaryServerInterceptor(authFunc(secret)),
			grpc_recovery.UnaryServerInterceptor(),
		)),
	}
}

func registerServices(g *grpc.Server) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	pb.RegisterDeployServiceServer(g, deploy.NewServer(wd))
	healthpb.RegisterHealthServer(g, health.NewServer())
}
