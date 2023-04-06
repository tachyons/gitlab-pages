package projectroot

import (
	"context"
	"fmt"
	"os"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

// Root implements the more low-level vfs.Root interface and can be used in its
// stead. The difference is, it always resolves the files inside the project's
// rootDirectory by prepending that dir to any open request.
type Root struct {
	rootDirectory string
	vfsRoot       vfs.Root
}

func (r *Root) getPath(name string) string {
	if r.rootDirectory == "" {
		// In case the GitLab API is not up-to-date this string may be empty.
		// In that case default to the legacy behavior
		r.rootDirectory = "public"
	}

	return fmt.Sprintf("%s/%s", r.rootDirectory, name)
}

func (r *Root) Open(ctx context.Context, name string) (vfs.File, error) {
	return r.vfsRoot.Open(ctx, r.getPath(name))
}

func (r *Root) Lstat(ctx context.Context, name string) (os.FileInfo, error) {
	return r.vfsRoot.Lstat(ctx, r.getPath(name))
}

func (r *Root) Readlink(ctx context.Context, name string) (string, error) {
	return r.vfsRoot.Readlink(ctx, r.getPath(name))
}

func New(rootDirectory string, vfsRoot vfs.Root) *Root {
	return &Root{
		rootDirectory: rootDirectory,
		vfsRoot:       vfsRoot,
	}
}
