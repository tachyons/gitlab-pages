package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type gitlabSourceConfig struct {
	Domains struct {
		Enabled []string
		Broken  string
	}
}

// WatchForGitlabSourceConfigChange polls the filesystem and updates test domains if needed.
func WatchForGitlabSourceConfigChange(gitlabSourceEnabledDomains *[]string, gitlabSourceBrokenDomain *string, interval time.Duration) {
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
			} else if len(*gitlabSourceEnabledDomains) > 1 || len(*gitlabSourceBrokenDomain) > 1 {
				*gitlabSourceEnabledDomains = []string{}
				*gitlabSourceBrokenDomain = ""
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

		config := gitlabSourceConfig{}
		err = yaml.Unmarshal(content, &config)
		if err != nil {
			log.WithError(err).Warn("Failed to decode gitlab source config file")

			time.Sleep(interval)
			continue
		}

		*gitlabSourceEnabledDomains = config.Domains.Enabled
		*gitlabSourceBrokenDomain = config.Domains.Broken
		// log.Info("gitlab source config updated")
		log.WithFields(log.Fields{
			"domains": *gitlabSourceEnabledDomains,
		}).Info("gitlab source config updated")

		time.Sleep(interval)
	}
}
