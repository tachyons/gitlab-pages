package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			gitLabServer:     "gitlabserver.com",
			gitLabAuthServer: "authserver.com",
			artifactsServer:  "https://artifactsserver.com",
			expected:         "gitlabserver.com",
		},
		{
			name:             "When auth server is set",
			gitLabServer:     "",
			gitLabAuthServer: "authserver.com",
			artifactsServer:  "https://artifactsserver.com",
			expected:         "authserver.com",
		},
		{
			name:             "When only artifacts server is set",
			gitLabServer:     "",
			gitLabAuthServer: "",
			artifactsServer:  "https://artifactsserver.com",
			expected:         "artifactsserver.com",
		},
		{
			name:             "When only artifacts server includes path",
			gitLabServer:     "",
			gitLabAuthServer: "",
			artifactsServer:  "https://artifactsserver.com:8080/api/path",
			expected:         "artifactsserver.com",
		}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gitLabServer = &test.gitLabServer
			gitLabAuthServer = &test.gitLabAuthServer
			artifactsServer = &test.artifactsServer
			assert.Equal(t, test.expected, gitlabServerFromFlags())
		})
	}
}
