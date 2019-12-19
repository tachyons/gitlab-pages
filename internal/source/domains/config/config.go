package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// GitlabSourceConfig holds the configuration for the gitlab source
type GitlabSourceConfig struct {
	Domains struct {
		Enabled []string
		Broken  string
	}
}

// Empty checks if the config is empty
func (config *GitlabSourceConfig) Empty() bool {
	enabledDomainsEmpty := len(config.Domains.Enabled) == 0
	brokenDomainEmpty := len(config.Domains.Broken) == 0

	return enabledDomainsEmpty && brokenDomainEmpty
}

// Reset wipes out any configuration already set
func (config *GitlabSourceConfig) Reset() {
	config.Domains.Enabled = []string{}
	config.Domains.Broken = ""
}

// UpdateFromYaml updates the config
// We use new variable here (instead of using `config` directly)
// because if `content` is empty `yaml.Unmarshal` does not update
// the fields already set.
func (config *GitlabSourceConfig) UpdateFromYaml(content []byte) error {
	updated := GitlabSourceConfig{}

	err := yaml.Unmarshal(content, &updated)
	if err != nil {
		return err
	}

	*config = updated

	return nil
}

// WatchForGitlabSourceConfigChange polls the filesystem and updates test domains if needed.
func WatchForGitlabSourceConfigChange(config *GitlabSourceConfig, interval time.Duration) {
	var lastContent []byte

	gitlabSourceConfigFile := os.Getenv("GITLAB_SOURCE_CONFIG_FILE")
	if gitlabSourceConfigFile == "" {
		gitlabSourceConfigFile = ".gitlab-source-config.yml"
	}

	for {
		content, err := ioutil.ReadFile(gitlabSourceConfigFile)

		if err == nil {
			if !bytes.Equal(lastContent, content) {
				lastContent = content

				err = config.UpdateFromYaml(content)
				if err != nil {
					log.WithError(err).Warn("Failed to decode gitlab source config file")
				} else {
					log.WithFields(log.Fields{
						"Enabled domains": config.Domains.Enabled,
						"Broken domain":   config.Domains.Broken,
					}).Info("gitlab source config updated")
				}
			}
		} else {
			if os.IsNotExist(err) {
				if !config.Empty() {
					config.Reset()
					lastContent = []byte{}
					log.Info("Config file removed, disabling gitlab source")
				}
			} else {
				log.WithError(err).Warn("Failed to read gitlab source config file")
			}
		}

		time.Sleep(interval)
	}
}
