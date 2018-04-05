package deploy

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	pb "gitlab.com/gitlab-org/gitlab-pages-proto/go"
)

type server struct{}

func (*server) DeleteSite(context.Context, *pb.DeleteSiteRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}
