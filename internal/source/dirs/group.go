package dirs

import (
	"errors"
	"net/http"
	"path"
	"strings"

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

type projects map[string]*ProjectConfig
type subgroups map[string]*Group

func (g *Group) digProjectWithSubpath(parentPath string, keys []string) (*ProjectConfig, string, string) {
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
func (group *Group) getProjectConfigWithSubpath(r *http.Request) (*ProjectConfig, string, string) {
	// Check for a project specified in the URL: http://group.gitlab.io/projectA
	// If present, these projects shadow the group domain.
	split := strings.SplitN(r.URL.Path, "/", maxProjectDepth)
	if len(split) >= 2 {
		projectConfig, projectPath, urlPath := group.digProjectWithSubpath("", split[1:])
		if projectConfig != nil {
			return projectConfig, projectPath, urlPath
		}
	}

	// Since the URL doesn't specify a project (e.g. http://mydomain.gitlab.io),
	// return the group project if it exists.
	if host := host.FromRequest(r); host != "" {
		if groupProject := group.projects[host]; groupProject != nil {
			return groupProject, host, strings.Join(split[1:], "/")
		}
	}

	return nil, "", ""
}

func (g *Group) IsHTTPSOnly(r *http.Request) (bool, error) {
	project, _, _ := g.getProjectConfigWithSubpath(r)

	if project != nil {
		return project.HTTPSOnly, nil
	}

	return false, errors.New("project not found")
}

func (g *Group) HasAccessControl(r *http.Request) (bool, error) {
	project, _, _ := g.getProjectConfigWithSubpath(r)

	if project != nil {
		return project.AccessControl, nil
	}

	return false, errors.New("project not found")
}

func (g *Group) IsNamespaceProject(r *http.Request) (bool, error) {
	project, _, _ := g.getProjectConfigWithSubpath(r)

	if project != nil {
		return project.NamespaceProject, nil
	}

	return false, errors.New("project not found")
}

func (g *Group) ProjectID(r *http.Request) (uint64, error) {
	project, _, _ := g.getProjectConfigWithSubpath(r)

	if project != nil {
		return project.ID, nil
	}

	return 0, errors.New("project not found")
}

func (g *Group) ProjectExists(r *http.Request) (bool, error) {
	project, _, _ := g.getProjectConfigWithSubpath(r)

	if project != nil {
		return true, nil
	}

	return false, nil
}

func (g *Group) ProjectWithSubpath(r *http.Request) (string, string, error) {
	project, projectName, subPath := g.getProjectConfigWithSubpath(r)

	if project != nil {
		return projectName, subPath, nil
	}

	return "", "", errors.New("project not found")
}
