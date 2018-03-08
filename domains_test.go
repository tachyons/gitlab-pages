package main

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// This write must be atomic, otherwise we cannot predict the state of the
// domain watcher goroutine. We cannot use ioutil.WriteFile because that
// has a race condition where the file is empty, which can get picked up
// by the domain watcher.
func writeRandomTimestamp(t *testing.T) {
	b := make([]byte, 10)
	n, _ := rand.Read(b)
	require.True(t, n > 0, "read some random bytes")

	temp, err := ioutil.TempFile(".", "TestWatchDomains")
	require.NoError(t, err)
	_, err = temp.Write(b)
	require.NoError(t, err, "write to tempfile")
	require.NoError(t, temp.Close(), "close tempfile")

	require.NoError(t, os.Rename(temp.Name(), updateFile), "rename tempfile")
}

func TestWatchDomains(t *testing.T) {
	setUpTests()

	require.NoError(t, os.RemoveAll(updateFile))

	update := make(chan domains)
	go watchDomains("gitlab.io", func(domains domains) {
		update <- domains
	}, time.Microsecond*50)

	defer os.Remove(updateFile)

	domains := recvTimeout(t, update)
	assert.NotNil(t, domains, "if the domains are fetched on start")

	writeRandomTimestamp(t)
	domains = recvTimeout(t, update)
	assert.NotNil(t, domains, "if the domains are updated after the creation")

	writeRandomTimestamp(t)
	domains = recvTimeout(t, update)
	assert.NotNil(t, domains, "if the domains are updated after the timestamp change")
}

func recvTimeout(t *testing.T, ch <-chan domains) domains {
	timeout := 5 * time.Second

	select {
	case d := <-ch:
		return d
	case <-time.After(timeout):
		t.Fatalf("timeout after %v waiting for domain update", timeout)
		return nil
	}
}

func BenchmarkReadGroups(b *testing.B) {
	testRoot, err := ioutil.TempDir("", "gitlab-pages-test")
	require.NoError(b, err)

	cwd, err := os.Getwd()
	require.NoError(b, err)

	defer func(oldWd, testWd string) {
		os.Chdir(oldWd)
		fmt.Printf("cleaning up test directory %s\n", testWd)
		os.RemoveAll(testWd)
	}(cwd, testRoot)

	require.NoError(b, os.Chdir(testRoot))

	nGroups := 10000
	b.Logf("creating fake domains directory with %d groups", nGroups)
	for i := 0; i < nGroups; i++ {
		for j := 0; j < 5; j++ {
			dir := fmt.Sprintf("%s/group-%d/project-%d", testRoot, i, j)
			require.NoError(b, os.MkdirAll(dir+"/public", 0755))

			fakeConfig := fmt.Sprintf(`{"Domains":[{"Domain":"foo.%d.%d.example.io","Certificate":"bar","Key":"baz"}]}`, i, j)
			require.NoError(b, ioutil.WriteFile(dir+"/config.json", []byte(fakeConfig), 0644))
		}
		if i%100 == 0 {
			fmt.Print(".")
		}
	}

	b.Run("ReadGroups", func(b *testing.B) {
		var testDomains domains
		for i := 0; i < 2; i++ {
			testDomains = domains(make(map[string]*domain))
			require.NoError(b, testDomains.ReadGroups("example.com"))
		}
		b.Logf("found %d domains", len(testDomains))
	})
}
