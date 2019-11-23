package disk

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/karrick/godirwalk"
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
	cleanup := setUpTests(t)
	defer cleanup()

	dm := make(Map)
	dm.ReadGroups("test.io", getEntries(t))

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
		"private.domain.com",
		"group.auth.test.io",
		"group.acme.test.io",
		"withacmechallenge.domain.com",
		"capitalgroup.test.io",
	}

	for _, expected := range domains {
		require.Contains(t, domains, expected)
	}

	for _, actual := range domains {
		require.Contains(t, expectedDomains, actual)
	}

	// Check that multiple domains in the same project are recorded faithfully
	require.Equal(t, "test.domain.com", dm["test.domain.com"].Name)
	require.Equal(t, "other.domain.com", dm["other.domain.com"].Name)
	require.Equal(t, "test", dm["other.domain.com"].CertificateCert)
	require.Equal(t, "key", dm["other.domain.com"].CertificateKey)

	// check subgroups
	domain, ok := dm["group.test.io"]
	require.True(t, ok, "missing group.test.io domain")
	subgroup, ok := domain.Resolver.(*Group).subgroups["subgroup"]
	require.True(t, ok, "missing group.test.io subgroup")
	_, ok = subgroup.projects["project"]
	require.True(t, ok, "missing project for subgroup in group.test.io domain")
}

func TestReadProjectsMaxDepth(t *testing.T) {
	nGroups := 3
	levels := subgroupScanLimit + 5
	cleanup := buildFakeDomainsDirectory(t, nGroups, levels)
	defer cleanup()

	defaultDomain := "test.io"
	dm := make(Map)
	dm.ReadGroups(defaultDomain, getEntries(t))

	var domains []string
	for d := range dm {
		domains = append(domains, d)
	}

	var expectedDomains []string
	for i := 0; i < nGroups; i++ {
		expectedDomains = append(expectedDomains, fmt.Sprintf("group-%d.%s", i, defaultDomain))
	}

	for _, expected := range domains {
		require.Contains(t, domains, expected)
	}

	for _, actual := range domains {
		// we are not checking config.json domains here
		if !strings.HasSuffix(actual, defaultDomain) {
			continue
		}
		require.Contains(t, expectedDomains, actual)
	}

	// check subgroups
	domain, ok := dm["group-0.test.io"]
	require.True(t, ok, "missing group-0.test.io domain")
	subgroup := domain.Resolver.(*Group)
	for i := 0; i < levels; i++ {
		subgroup, ok = subgroup.subgroups["sub"]
		if i <= subgroupScanLimit {
			require.True(t, ok, "missing group-0.test.io subgroup at level %d", i)
			_, ok = subgroup.projects["project-0"]
			require.True(t, ok, "missing project for subgroup in group-0.test.io domain at level %d", i)
		} else {
			require.False(t, ok, "subgroup level %d. Maximum allowed nesting level is %d", i, subgroupScanLimit)
			break
		}
	}
}

// This write must be atomic, otherwise we cannot predict the state of the
// domain watcher goroutine. We cannot use ioutil.WriteFile because that
// has a race condition where the file is empty, which can get picked up
// by the domain watcher.
func writeRandomTimestamp(t *testing.T) {
	b := make([]byte, 10)
	n, _ := rand.Read(b)
	require.True(t, n > 0, "read some random bytes")

	temp, err := ioutil.TempFile(".", "TestIsWatch")
	require.NoError(t, err)
	_, err = temp.Write(b)
	require.NoError(t, err, "write to tempfile")
	require.NoError(t, temp.Close(), "close tempfile")

	require.NoError(t, os.Rename(temp.Name(), updateFile), "rename tempfile")
}

func TestWatch(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	require.NoError(t, os.RemoveAll(updateFile))

	update := make(chan Map)
	go Watch("gitlab.io", func(dm Map) {
		update <- dm
	}, time.Microsecond*50)

	defer os.Remove(updateFile)

	domains := recvTimeout(t, update)
	require.NotNil(t, domains, "if the domains are fetched on start")

	writeRandomTimestamp(t)
	domains = recvTimeout(t, update)
	require.NotNil(t, domains, "if the domains are updated after the creation")

	writeRandomTimestamp(t)
	domains = recvTimeout(t, update)
	require.NotNil(t, domains, "if the domains are updated after the timestamp change")
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

func buildFakeDomainsDirectory(t require.TestingT, nGroups, levels int) func() {
	testRoot, err := ioutil.TempDir("", "gitlab-pages-test")
	require.NoError(t, err)

	for i := 0; i < nGroups; i++ {
		parent := fmt.Sprintf("%s/group-%d", testRoot, i)
		domain := fmt.Sprintf("%d.example.io", i)
		buildFakeProjectsDirectory(t, parent, domain)
		for j := 0; j < levels; j++ {
			parent = fmt.Sprintf("%s/sub", parent)
			domain = fmt.Sprintf("%d.%s", j, domain)
			buildFakeProjectsDirectory(t, parent, domain)
		}
		if i%100 == 0 {
			fmt.Print(".")
		}
	}

	cleanup := chdirInPath(t, testRoot)

	return func() {
		defer cleanup()
		fmt.Printf("cleaning up test directory %s\n", testRoot)
		os.RemoveAll(testRoot)
	}
}

func buildFakeProjectsDirectory(t require.TestingT, groupPath, domain string) {
	for j := 0; j < 5; j++ {
		dir := fmt.Sprintf("%s/project-%d", groupPath, j)
		require.NoError(t, os.MkdirAll(dir+"/public", 0755))

		fakeConfig := fmt.Sprintf(`{"Domains":[{"Domain":"foo.%d.%s","Certificate":"bar","Key":"baz"}]}`, j, domain)
		require.NoError(t, ioutil.WriteFile(dir+"/config.json", []byte(fakeConfig), 0644))
	}
}

func BenchmarkReadGroups(b *testing.B) {
	nGroups := 10000
	b.Logf("creating fake domains directory with %d groups", nGroups)
	cleanup := buildFakeDomainsDirectory(b, nGroups, 0)
	defer cleanup()

	b.Run("ReadGroups", func(b *testing.B) {
		var dm Map
		for i := 0; i < 2; i++ {
			dm = make(Map)
			dm.ReadGroups("example.com", getEntriesForBenchmark(b))
		}
		b.Logf("found %d domains", len(dm))
	})
}
