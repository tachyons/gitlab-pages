package domain

import (
	"path"
	"strings"
)

type projects map[string]*project
type subgroups map[string]*group

type group struct {
	name string

	// nested groups
	subgroups subgroups

	// group domains:
	projects projects
}

func (g *group) digProjectWithSubpath(parentPath string, keys []string) (*project, string, string) {
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
