package main

import (
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestReadProjects(t *testing.T) {
	*pagesDomain = "test.io"

	d := make(domains)
	err := d.ReadGroups()
	require.NoError(t, err)

	var domains []string
	for domain, _ := range d {
		domains = append(domains, domain)
	}

	expectedDomains := []string{
		"group.test.io",
		"group.internal.test.io",
		"test.domain.com", // from config.json
		"other.domain.com",
	}

	for _, expected := range domains {
		assert.Contains(t, domains, expected)
	}

	for _, actual := range domains {
		assert.Contains(t, expectedDomains, actual)
	}
}
