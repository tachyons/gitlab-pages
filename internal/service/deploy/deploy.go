package deploy

import (
	"github.com/golang/protobuf/ptypes/empty"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
	"golang.org/x/net/context"
)

type server struct{}

func (*server) DeleteSite(context.Context, *pb.DeleteSiteRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
