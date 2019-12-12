package source

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
)

type gitlabSourceConfig struct {
	Domains struct {
		Enabled []string
		Broken  string
	}
}

var gitlabSourceDomains []string
var brokenSourceDomain string

func init() {
	go watchForGitlabSourceConfigChange(&gitlabSourceDomains, &brokenSourceDomain, 5*time.Second)
}

// watchForGitlabSourceConfigChange polls the filesystem and updates test domains if needed.
func watchForGitlabSourceConfigChange(gitlabSourceEnabledDomains *[]string, gitlabSourceBrokenDomain *string, interval time.Duration) {
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
		log.Info("gitlab source config updated")

		time.Sleep(interval)
	}
}

// Domains struct represents a map of all domains supported by pages. It is
// currently using two sources during the transition to the new GitLab domains
// source.
type Domains struct {
	gitlab Source
	disk   *disk.Disk // legacy disk source
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(config Config) (*Domains, error) {
	if len(config.GitlabServerURL()) == 0 || len(config.GitlabAPISecret()) == 0 {
		return &Domains{disk: disk.New()}, nil
	}

	gitlab, err := gitlab.New(config)
	if err != nil {
		return nil, err
	}

	return &Domains{
		gitlab: gitlab,
		disk:   disk.New(),
	}, nil
}

// GetDomain retrieves a domain information from a source. We are using two
// sources here because it allows us to switch behavior and the domain source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk source.
func (d *Domains) GetDomain(name string) (*domain.Domain, error) {
	if name == brokenSourceDomain {
		return nil, errors.New("broken test domain used")
	}

	return d.source(name).GetDomain(name)
}

// Read starts the disk domain source. It is DEPRECATED, because we want to
// remove it entirely when disk source gets removed.
func (d *Domains) Read(rootDomain string) {
	d.disk.Read(rootDomain)
}

// IsReady checks if the disk domain source managed to traverse entire pages
// filesystem and is ready for use. It is DEPRECATED, because we want to remove
// it entirely when disk source gets removed.
func (d *Domains) IsReady() bool {
	return d.disk.IsReady()
}

func (d *Domains) source(domain string) Source {
	if d.gitlab == nil {
		return d.disk
	}

	for _, name := range gitlabSourceDomains {
		if domain == name {
			return d.gitlab
		}
	}

	return d.disk
}
