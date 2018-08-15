package domain

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/karrick/godirwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getEntries(t *testing.T) godirwalk.Dirents {
	fis, err := godirwalk.ReadDirents(".", nil)

	require.NoError(t, err)

	return fis
}

func getEntriesForBenchmark(t *testing.B) godirwalk.Dirents {
	fis, err := godirwalk.ReadDirents(".", nil)

	require.NoError(t, err)

	return fis
}

func TestReadProjects(t *testing.T) {
	setUpTests()

	dm := make(Map)
	err := dm.ReadGroups("test.io", getEntries(t))
	require.NoError(t, err)

	var domains []string
	for d := range dm {
		domains = append(domains, d)
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
	assert.Equal(t, exp1, dm["test.domain.com"].config)

	exp2 := &domainConfig{Domain: "other.domain.com", Certificate: "test", Key: "key"}
	assert.Equal(t, exp2, dm["other.domain.com"].config)
}

// This write must be atomic, otherwise we cannot predict the state of the
// domain watcher goroutine. We cannot use ioutil.WriteFile because that
// has a race condition where the file is empty, which can get picked up
// by the domain watcher.
func writeRandomTimestamp(t *testing.T) {
	b := make([]byte, 10)
	n, _ := rand.Read(b)
	require.True(t, n > 0, "read some random bytes")

	temp, err := ioutil.TempFile(".", "TestWatch")
	require.NoError(t, err)
	_, err = temp.Write(b)
	require.NoError(t, err, "write to tempfile")
	require.NoError(t, temp.Close(), "close tempfile")

	require.NoError(t, os.Rename(temp.Name(), updateFile), "rename tempfile")
}

func TestWatch(t *testing.T) {
	setUpTests()

	require.NoError(t, os.RemoveAll(updateFile))

	update := make(chan Map)
	go Watch("gitlab.io", func(dm Map) {
		update <- dm
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

func recvTimeout(t *testing.T, ch <-chan Map) Map {
	timeout := 5 * time.Second

	select {
	case dm := <-ch:
		return dm
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
		var dm Map
		for i := 0; i < 2; i++ {
			dm = make(Map)
			require.NoError(b, dm.ReadGroups("example.com", getEntriesForBenchmark(b)))
		}
		b.Logf("found %d domains", len(dm))
	})
}
