package domain

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const configFile = "test-group/test-project/config.json"
const invalidConfig = `{"Domains":{}}`
const validConfig = `{"Domains":[{"Domain":"test"}]}`

// temporary type alias
type domainsConfig = legacyDomainsConfig
type domainConfig = Config

func TestDomainConfigValidness(t *testing.T) {
	d := domainConfig{}
	require.False(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test"}
	require.True(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test"}
	require.True(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.gitlab.io"}
	require.False(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.test.gitlab.io"}
	require.False(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.testgitlab.io"}
	require.True(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.GitLab.Io"}
	require.False(t, d.Valid("gitlab.io"))
}

func TestDomainConfigRead(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	d := domainsConfig{}
	err := d.Read("test-group", "test-project")
	require.Error(t, err)

	os.MkdirAll(filepath.Dir(configFile), 0700)
	defer os.RemoveAll("test-group")

	d = domainsConfig{}
	err = d.Read("test-group", "test-project")
	require.Error(t, err)

	err = ioutil.WriteFile(configFile, []byte(invalidConfig), 0600)
	require.NoError(t, err)
	d = domainsConfig{}
	err = d.Read("test-group", "test-project")
	require.Error(t, err)

	err = ioutil.WriteFile(configFile, []byte(validConfig), 0600)
	require.NoError(t, err)
	d = domainsConfig{}
	err = d.Read("test-group", "test-project")
	require.NoError(t, err)
}
