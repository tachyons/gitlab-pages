package main

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

const updateFile = ".update"

func TestReadProjects(t *testing.T) {
	setUpTests()

	d := make(domains)
	err := d.ReadGroups("test.io")
	require.NoError(t, err)

	var domains []string
	for domain := range d {
		domains = append(domains, domain)
	}

	expectedDomains := []string{
		"group.test.io",
		"group.internal.test.io",
		"test.domain.com", // from config.json
		"other.domain.com",
		"domain.404.com",
		"group.404.test.io",
	}

	for _, expected := range domains {
		assert.Contains(t, domains, expected)
	}

	for _, actual := range domains {
		assert.Contains(t, expectedDomains, actual)
	}
}

func writeRandomTimestamp() {
	b := make([]byte, 10)
	rand.Read(b)
	ioutil.WriteFile(updateFile, b, 0600)
}

func TestWatchDomains(t *testing.T) {
	setUpTests()

	update := make(chan domains)
	lastUpdate := []byte("no-update")
	go watchDomains("gitlab.io", func(domains domains) {
		update <- domains
	}, time.Microsecond*50, &lastUpdate)

	defer os.Remove(updateFile)

	domains := <-update
	assert.NotNil(t, domains, "if the domains are fetched on start")

	writeRandomTimestamp()
	domains = <-update
	assert.NotNil(t, domains, "if the domains are updated after the creation")

	writeRandomTimestamp()
	domains = <-update
	assert.NotNil(t, domains, "if the domains are updated after the timestamp change")

	os.Remove(updateFile)
	domains = <-update
	assert.NotNil(t, domains, "if the domains are updated after the timestamp removal")
}
