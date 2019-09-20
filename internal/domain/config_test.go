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

func TestDomainConfigValidness(t *testing.T) {
	d := Config{}
	require.False(t, d.Valid("gitlab.io"))

	d = Config{Domain: "test"}
	require.True(t, d.Valid("gitlab.io"))

	d = Config{Domain: "test"}
	require.True(t, d.Valid("gitlab.io"))

	d = Config{Domain: "test.gitlab.io"}
	require.False(t, d.Valid("gitlab.io"))

	d = Config{Domain: "test.test.gitlab.io"}
	require.False(t, d.Valid("gitlab.io"))

	d = Config{Domain: "test.testgitlab.io"}
	require.True(t, d.Valid("gitlab.io"))

	d = Config{Domain: "test.GitLab.Io"}
	require.False(t, d.Valid("gitlab.io"))
}

func TestDomainConfigRead(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	d := MultiConfig{}
	err := d.Read("test-group", "test-project")
	require.Error(t, err)

	os.MkdirAll(filepath.Dir(configFile), 0700)
	defer os.RemoveAll("test-group")

	d = MultiConfig{}
	err = d.Read("test-group", "test-project")
	require.Error(t, err)

	err = ioutil.WriteFile(configFile, []byte(invalidConfig), 0600)
	require.NoError(t, err)
	d = MultiConfig{}
	err = d.Read("test-group", "test-project")
	require.Error(t, err)

	err = ioutil.WriteFile(configFile, []byte(validConfig), 0600)
	require.NoError(t, err)
	d = MultiConfig{}
	err = d.Read("test-group", "test-project")
	require.NoError(t, err)
}
