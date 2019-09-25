package disk

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupDig(t *testing.T) {
	matchingProject := &ProjectConfig{ID: 1}

	tests := []struct {
		name                string
		g                   Group
		path                string
		expectedProject     *ProjectConfig
		expectedProjectPath string
		expectedPath        string
	}{
		{
			name: "empty group",
			path: "projectb/demo/features.html",
			g:    Group{},
		},
		{
			name: "group with project",
			path: "projectb/demo/features.html",
			g: Group{
				projects: projects{"projectb": matchingProject},
			},
			expectedProject:     matchingProject,
			expectedProjectPath: "projectb",
			expectedPath:        "demo/features.html",
		},
		{
			name: "group with project and no path in URL",
			path: "projectb",
			g: Group{
				projects: projects{"projectb": matchingProject},
			},
			expectedProject:     matchingProject,
			expectedProjectPath: "projectb",
		},
		{
			name: "group with subgroup and project",
			path: "projectb/demo/features.html",
			g: Group{
				projects: projects{"projectb": matchingProject},
				subgroups: subgroups{
					"sub1": &Group{
						projects: projects{"another": &ProjectConfig{}},
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
			g: Group{
				subgroups: subgroups{
					"sub1": &Group{
						projects: projects{"projectb": matchingProject},
					},
				},
				projects: projects{"another": &ProjectConfig{}},
			},
			expectedProject:     matchingProject,
			expectedProjectPath: "sub1/projectb",
			expectedPath:        "demo/features.html",
		},
		{
			name: "group with matching subgroup but no project",
			path: "sub1/projectb/demo/features.html",
			g: Group{
				subgroups: subgroups{
					"sub1": &Group{
						projects: projects{"another": &ProjectConfig{}},
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
