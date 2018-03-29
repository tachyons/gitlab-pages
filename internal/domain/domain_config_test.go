package domain

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const configFile = "test-group/test-project/config.json"
const invalidConfig = `{"Domains":{}}`
const validConfig = `{"Domains":[{"Domain":"test"}]}`

func TestDomainConfigValidness(t *testing.T) {
	d := domainConfig{}
	assert.False(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test"}
	assert.True(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test"}
	assert.True(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.gitlab.io"}
	assert.False(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.test.gitlab.io"}
	assert.False(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.testgitlab.io"}
	assert.True(t, d.Valid("gitlab.io"))

	d = domainConfig{Domain: "test.GitLab.Io"}
	assert.False(t, d.Valid("gitlab.io"))
}

func TestDomainConfigRead(t *testing.T) {
	setUpTests()

	d := domainsConfig{}
	err := d.Read("test-group", "test-project")
	assert.Error(t, err)

	os.MkdirAll(filepath.Dir(configFile), 0700)
	defer os.RemoveAll("test-group")

	d = domainsConfig{}
	err = d.Read("test-group", "test-project")
	assert.Error(t, err)

	err = ioutil.WriteFile(configFile, []byte(invalidConfig), 0600)
	require.NoError(t, err)
	d = domainsConfig{}
	err = d.Read("test-group", "test-project")
	assert.Error(t, err)

	err = ioutil.WriteFile(configFile, []byte(validConfig), 0600)
	require.NoError(t, err)
	d = domainsConfig{}
	err = d.Read("test-group", "test-project")
	require.NoError(t, err)
}
