package main

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"group.https-only.test.io",
		"test.my-domain.com",
		"test2.my-domain.com",
		"no.cert.com",
	}

	for _, expected := range domains {
		assert.Contains(t, domains, expected)
	}

	for _, actual := range domains {
		assert.Contains(t, expectedDomains, actual)
	}

	// Check that multiple domains in the same project are recorded faithfully
	exp1 := &domainConfig{Domain: "test.domain.com"}
	assert.Equal(t, exp1, d["test.domain.com"].Config)

	exp2 := &domainConfig{Domain: "other.domain.com", Certificate: "test", Key: "key"}
	assert.Equal(t, exp2, d["other.domain.com"].Config)
}

func writeRandomTimestamp() {
	b := make([]byte, 10)
	rand.Read(b)
	ioutil.WriteFile(updateFile, b, 0600)
}

func TestWatchDomains(t *testing.T) {
	setUpTests()

	update := make(chan domains)
	go watchDomains("gitlab.io", func(domains domains) {
		update <- domains
	}, time.Microsecond*50)

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
