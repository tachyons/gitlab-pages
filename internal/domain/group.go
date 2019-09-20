package domain

import (
	"path"
	"strings"
)

// Group represents a GitLab group with projects and subgroups
type Group struct {
	name string

	// nested groups
	subgroups subgroups

	// group domains:
	projects projects
}

type projects map[string]*Project
type subgroups map[string]*Group

func (g *Group) digProjectWithSubpath(parentPath string, keys []string) (*Project, string, string) {
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
