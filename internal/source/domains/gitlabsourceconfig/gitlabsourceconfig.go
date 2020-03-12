package gitlabsourceconfig

import (
	"bytes"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// GitlabSourceDomains holds the domains to be used with the gitlab source
type GitlabSourceDomains struct {
	Enabled []string
	Broken  string
	Rollout GitlabSourceRollout
}

// GitlabSourceRollout holds the rollout strategy and percentage
type GitlabSourceRollout struct {
	Stickiness string
	Percentage int
}

// GitlabSourceConfig holds the configuration for the gitlab source
type GitlabSourceConfig struct {
	Domains GitlabSourceDomains
}

// UpdateFromYaml updates the config
// We use new variable here (instead of using `config` directly)
// because if `content` is empty `yaml.Unmarshal` does not update
// the fields already set.
func (config *GitlabSourceConfig) UpdateFromYaml(content []byte) error {
	updated := GitlabSourceConfig{}
	log.Infof("we should be coming here....")
	err := yaml.Unmarshal(content, &updated)
	if err != nil {
		log.WithError(err).Error("oooops...")

		return err
	}

	*config = updated

	log.WithFields(log.Fields{
		"Enabled domains":    config.Domains.Enabled,
		"Broken domain":      config.Domains.Broken,
		"Rollout %":          config.Domains.Rollout.Percentage,
		"Rollout stickiness": config.Domains.Rollout.Stickiness,
	}).Info("gitlab source config updated")

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
		content, err := readConfig(gitlabSourceConfigFile)
		if err != nil {
			log.WithError(err).Warn("Failed to read gitlab source config file")

			time.Sleep(interval)
			continue
		}

		if !bytes.Equal(lastContent, content) {
			lastContent = content

			err = config.UpdateFromYaml(content)
			if err != nil {
				log.WithError(err).Warn("Failed to update gitlab source config")
			}
		}

		time.Sleep(interval)
	}
}

func readConfig(configfile string) ([]byte, error) {
	content, err := ioutil.ReadFile(configfile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return content, nil
}
