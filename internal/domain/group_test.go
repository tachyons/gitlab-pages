package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupDig(t *testing.T) {
	matchingProject := &project{ID: 1}

	tests := []struct {
		name                string
		g                   group
		path                string
		expectedProject     *project
		expectedProjectPath string
		expectedPath        string
	}{
		{
			name: "empty group",
			path: "projectb/demo/features.html",
			g:    group{},
		},
		{
			name: "group with project",
			path: "projectb/demo/features.html",
			g: group{
				projects: projects{"projectb": matchingProject},
			},
			expectedProject:     matchingProject,
			expectedProjectPath: "projectb",
			expectedPath:        "demo/features.html",
		},
		{
			name: "group with project and no path in URL",
			path: "projectb",
			g: group{
				projects: projects{"projectb": matchingProject},
			},
			expectedProject:     matchingProject,
			expectedProjectPath: "projectb",
		},
		{
			name: "group with subgroup and project",
			path: "projectb/demo/features.html",
			g: group{
				projects: projects{"projectb": matchingProject},
				subgroups: subgroups{
					"sub1": &group{
						projects: projects{"another": &project{}},
					},
				},
			},
			expectedProject:     matchingProject,
			expectedProjectPath: "projectb",
			expectedPath:        "demo/features.html",
		},
		{
			name: "group with project inside a subgroup",
			path: "sub1/projectb/demo/features.html",
			g: group{
				subgroups: subgroups{
					"sub1": &group{
						projects: projects{"projectb": matchingProject},
					},
				},
				projects: projects{"another": &project{}},
			},
			expectedProject:     matchingProject,
			expectedProjectPath: "sub1/projectb",
			expectedPath:        "demo/features.html",
		},
		{
			name: "group with matching subgroup but no project",
			path: "sub1/projectb/demo/features.html",
			g: group{
				subgroups: subgroups{
					"sub1": &group{
						projects: projects{"another": &project{}},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, projectPath, urlPath := test.g.digProjectWithSubpath("", strings.Split(test.path, "/"))

			require.Equal(t, test.expectedProject, project)
			require.Equal(t, test.expectedProjectPath, projectPath)
			require.Equal(t, test.expectedPath, urlPath)
		})
	}
}
