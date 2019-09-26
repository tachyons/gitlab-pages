package disk

import (
	"net/http"
	"path"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/host"
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
func (g *Group) getProjectConfigWithSubpath(r *http.Request) (*projectConfig, string, string) {
	// Check for a project specified in the URL: http://group.gitlab.io/projectA
	// If present, these projects shadow the group domain.
	split := strings.SplitN(r.URL.Path, "/", maxProjectDepth)
	if len(split) >= 2 {
		projectConfig, projectPath, urlPath := g.digProjectWithSubpath("", split[1:])
		if projectConfig != nil {
			return projectConfig, projectPath, urlPath
		}
	}

	// Since the URL doesn't specify a project (e.g. http://mydomain.gitlab.io),
	// return the group project if it exists.
	if host := host.FromRequest(r); host != "" {
		if groupProject := g.projects[host]; groupProject != nil {
			return groupProject, host, strings.Join(split[1:], "/")
		}
	}

	return nil, "", ""
}

// Resolve tries to find project and its config recursively for a given request
// to a group domain
func (g *Group) Resolve(r *http.Request) (*domain.Project, string, error) {
	projectConfig, projectPath, subPath := g.getProjectConfigWithSubpath(r)

	if projectConfig == nil {
		return nil, "", nil // it is not an error when project does not exist
	}

	project := &domain.Project{
		LookupPath:         projectPath,
		IsNamespaceProject: projectConfig.NamespaceProject,
		IsHTTPSOnly:        projectConfig.HTTPSOnly,
		HasAccessControl:   projectConfig.AccessControl,
		ID:                 projectConfig.ID,
	}

	return project, subPath, nil
}
