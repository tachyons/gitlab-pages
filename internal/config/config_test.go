package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitLabServerFromFlags(t *testing.T) {
	tests := []struct {
		name             string
		gitLabServer     string
		gitLabAuthServer string
		artifactsServer  string
		expected         string
	}{
		{
			name:             "When gitLabServer is set",
			gitLabServer:     "https://gitlabserver.com",
			gitLabAuthServer: "https://authserver.com",
			artifactsServer:  "https://artifactsserver.com",
			expected:         "https://gitlabserver.com",
		},
		{
			name:             "When auth server is set",
			gitLabServer:     "",
			gitLabAuthServer: "https://authserver.com",
			artifactsServer:  "https://artifactsserver.com",
			expected:         "https://authserver.com",
		},
		{
			name:             "When only artifacts server is set",
			gitLabServer:     "",
			gitLabAuthServer: "",
			artifactsServer:  "https://artifactsserver.com",
			expected:         "https://artifactsserver.com",
		},
		{
			name:             "When only artifacts server includes path",
			gitLabServer:     "",
			gitLabAuthServer: "",
			artifactsServer:  "https://artifactsserver.com:8080/api/path",
			expected:         "https://artifactsserver.com:8080",
		}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gitLabServer = &test.gitLabServer
			gitLabAuthServer = &test.gitLabAuthServer
			artifactsServer = &test.artifactsServer
			gServer, err := gitlabServerFromFlags()
			require.NoError(t, err)
			require.Equal(t, test.expected, gServer)
		})
	}
}
