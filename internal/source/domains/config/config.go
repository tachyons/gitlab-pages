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

// WatchForGitlabSourceConfigChange polls the filesystem and updates test domains if needed.
func WatchForGitlabSourceConfigChange(config *GitlabSourceConfig, interval time.Duration) {
	var lastContent []byte

	gitlabSourceConfigFile := os.Getenv("GITLAB_SOURCE_CONFIG_FILE")
	if gitlabSourceConfigFile == "" {
		gitlabSourceConfigFile = ".gitlab-source-config.yml"
	}

	for {
		content, err := ioutil.ReadFile(gitlabSourceConfigFile)
		if err != nil {
			if !os.IsNotExist(err) {
				log.WithError(err).Warn("Failed to read gitlab source config file")
			} else if len(config.Domains.Enabled) > 1 || len(config.Domains.Broken) > 1 {
				config.Domains.Enabled = []string{}
				config.Domains.Broken = ""
				lastContent = []byte{}
				log.Info("Config file removed, disabling gitlab source")
			}

			time.Sleep(interval)
			continue
		}

		if bytes.Equal(lastContent, content) {
			time.Sleep(interval)
			continue
		}

		lastContent = content

		err = yaml.Unmarshal(content, config)
		if err != nil {
			log.WithError(err).Warn("Failed to decode gitlab source config file")

			time.Sleep(interval)
			continue
		}

		log.WithFields(log.Fields{
			"Enabled domains": config.Domains.Enabled,
			"Broken domain":   config.Domains.Broken,
		}).Info("👉 gitlab source config updated")

		time.Sleep(interval)
	}
}
