package deploy

import (
	"os"
	"regexp"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct{}

// NewServer returns a new deploy service server.
func NewServer() pb.DeployServiceServer {
	return &server{}
}

var traversalRegex = regexp.MustCompile(`(^\.\./)|(/\.\./)|(/\.\.$)`)

func (s *server) DeleteSite(ctx context.Context, req *pb.DeleteSiteRequest) (*empty.Empty, error) {
	if err := validatePath(req.Path); err != nil {
		return nil, err
	}

	st, err := os.Stat(req.Path)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "request.Path: %v", err)
	}
	if !st.IsDir() {
		return nil, status.Errorf(codes.FailedPrecondition, "not a directory: %q", req.Path)
	}

	return &empty.Empty{}, os.RemoveAll(req.Path)
}

func validatePath(requestPath string) error {
	if requestPath == "" {
		return status.Errorf(codes.InvalidArgument, "path empty")
	}

	if traversalRegex.MatchString(requestPath) {
		return status.Errorf(codes.InvalidArgument, "invalid path: %q", requestPath)
	}

	if strings.HasPrefix(requestPath, ".") || strings.HasPrefix(requestPath, "/") {
		return status.Errorf(codes.InvalidArgument, "invalid path: %q", requestPath)
	}

	return nil
}
