package deploy

import (
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	rootDir string
}

var traversalRegex = regexp.MustCompile(`(^\.\./)|(/\.\./)|(/\.\.$)`)

func (s *server) DeleteSite(ctx context.Context, req *pb.DeleteSiteRequest) (*empty.Empty, error) {
	if req.Path == "" {
		return nil, status.Errorf(codes.InvalidArgument, "path empty")
	}

	if traversalRegex.MatchString(req.Path) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid path: %q", req.Path)
	}

	if strings.HasPrefix(req.Path, ".") {
		return nil, status.Errorf(codes.InvalidArgument, "invalid path: %q", req.Path)
	}

	siteDir := path.Join(s.rootDir, req.Path)
	return &empty.Empty{}, os.RemoveAll(siteDir)
}
