package disk

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/host"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
)

const (
	subgroupScanLimit int = 21
	// maxProjectDepth is set to the maximum nested project depth in gitlab (21) plus 3.
	// One for the project, one for the first empty element of the split (URL.Path starts with /),
	// and one for the real file path
	maxProjectDepth int = subgroupScanLimit + 3
)

// Group represents a GitLab group with project configs and subgroups
type Group struct {
	name string

	// nested groups
	subgroups subgroups

	// group domains:
	projects projects
}

type projects map[string]*projectConfig
type subgroups map[string]*Group

func (g *Group) digProjectWithSubpath(parentPath string, keys []string) (*projectConfig, string, string) {
	if len(keys) >= 1 {
		head := keys[0]
		tail := keys[1:]
		currentPath := path.Join(parentPath, head)
		search := strings.ToLower(head)

		if project := g.projects[search]; project != nil {
			return project, currentPath, path.Join(tail...)
		}

		if subgroup := g.subgroups[search]; subgroup != nil {
			return subgroup.digProjectWithSubpath(currentPath, tail)
		}
	}

	return nil, "", ""
}

// Look up a project inside the domain based on the host and path. Returns the
// project and its name (if applicable)
func (g *Group) getProjectConfigWithSubpath(r *http.Request) (*projectConfig, string, string, string) {
	// Check for a project specified in the URL: http://group.gitlab.io/projectA
	// If present, these projects shadow the group domain.
	split := strings.SplitN(r.URL.Path, "/", maxProjectDepth)
	if len(split) >= 2 {
		projectConfig, projectPath, urlPath := g.digProjectWithSubpath("", split[1:])
		if projectConfig != nil {
			return projectConfig, "/" + projectPath, projectPath, urlPath
		}
	}

	// Since the URL doesn't specify a project (e.g. http://mydomain.gitlab.io),
	// return the group project if it exists.
	if host := host.FromRequest(r); host != "" {
		if groupProject := g.projects[host]; groupProject != nil {
			return groupProject, "/", host, strings.Join(split[1:], "/")
		}
	}

	return nil, "", "", ""
}

// Resolve tries to find project and its config recursively for a given request
// to a group domain
func (g *Group) Resolve(r *http.Request) (*serving.Request, error) {
	projectConfig, prefix, projectPath, subPath := g.getProjectConfigWithSubpath(r)

	if projectConfig == nil {
		// it is not an error when project does not exist, in that case
		// serving.Request.LookupPath is nil.
		return &serving.Request{Serving: disk.Instance()}, nil
	}

	lookupPath := &serving.LookupPath{
		VFS:                "local",
		Prefix:             prefix,
		Path:               filepath.Join(g.name, projectPath, "public") + "/",
		IsNamespaceProject: projectConfig.NamespaceProject,
		IsHTTPSOnly:        projectConfig.HTTPSOnly,
		HasAccessControl:   projectConfig.AccessControl,
		ProjectID:          projectConfig.ID,
	}

	lookupPath.VFS = "zip"
	lookupPath.Path = "http://192.168.88.233:9000/test-bucket/doc-gitlab-com.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=TEST_KEY%2F20200818%2F%2Fs3%2Faws4_request&X-Amz-Date=20200818T173935Z&X-Amz-Expires=432000&X-Amz-SignedHeaders=host&X-Amz-Signature=95810918d1b2441a07385838ebba5a0f01fdf4dcdf94ea9c602f8e7d06c84019"

	return &serving.Request{
		Serving:    disk.Instance(),
		LookupPath: lookupPath,
		SubPath:    subPath,
	}, nil
}
