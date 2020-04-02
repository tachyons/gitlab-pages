package source

import (
	"errors"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/rollout"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/domains/gitlabsourceconfig"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

var (
	gitlabSourceConfig gitlabsourceconfig.GitlabSourceConfig

	// serverlessDomainRegex is a regular expression we use to check if a domain
	// is a serverless domain, to short circut gitlab Source rollout. It can be
	// removed after the rollout is done
	serverlessDomainRegex = regexp.MustCompile(`^[^.]+-[[:xdigit:]]{2}a1[[:xdigit:]]{10}f2[[:xdigit:]]{2}[[:xdigit:]]+-?.*`)
)

func init() {
	// Start watching the config file for domains that will use the new `gitlab` Source,
	// to be removed once we switch completely to using it.
	go gitlabsourceconfig.WatchForGitlabSourceConfigChange(&gitlabSourceConfig, 1*time.Minute)
}

// Domains struct represents a map of all domains supported by pages. It is
// currently using two sources during the transition to the new GitLab domains
// Source.
type Domains struct {
	gitlab Source
	disk   *disk.Disk // legacy disk Source
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(config Config) (*Domains, error) {

	// fallback to disk if these values are empty
	// TODO communicate disk source deprecation https://gitlab.com/gitlab-org/gitlab-pages/-/issues/351
	if !config.GitlabEnableSourceAPI() || len(config.GitlabServerURL()) == 0 || len(config.GitlabAPISecret()) == 0 {
		log.Warn("disk source will be deprecated soon, please use -enable-domain-source-api")
		return &Domains{disk: disk.New()}, nil
	}

	gl, err := gitlab.New(config)
	if err != nil {
		if strings.Contains(err.Error(), client.ConnectionErrorMsg) {
			log.WithError(err).Warn("cannot -enable-domain-source-api, defaulting to disk source")
			return &Domains{disk: disk.New()}, nil
		}
		return nil, err
	}

	return &Domains{
		gitlab: gl,
		disk:   disk.New(),
	}, nil
}

// GetDomain retrieves a domain information from a Source. We are using two
// sources here because it allows us to switch behavior and the domain Source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk Source.
func (d *Domains) GetDomain(name string) (*domain.Domain, error) {
	if name == gitlabSourceConfig.Domains.Broken {
		return nil, errors.New("broken test domain used")
	}

	return d.source(name).GetDomain(name)
}

// Read starts the disk domain Source. It is DEPRECATED, because we want to
// remove it entirely when disk Source gets removed.
func (d *Domains) Read(rootDomain string) {
	d.disk.Read(rootDomain)
}

// IsReady checks if the disk domain Source managed to traverse entire pages
// filesystem and is ready for use. It is DEPRECATED, because we want to remove
// it entirely when disk Source gets removed.
func (d *Domains) IsReady() bool {
	return d.disk.IsReady()
}

func (d *Domains) source(domain string) Source {
	if d.gitlab == nil {
		return d.disk
	}

	// This check is only needed until we enable `d.gitlab` Source in all
	// environments (including on-premises installations) followed by removal of
	// `d.disk` Source. This can be safely removed afterwards.
	if IsServerlessDomain(domain) {
		return d.gitlab
	}

	for _, name := range gitlabSourceConfig.Domains.Enabled {
		if domain == name {
			return d.gitlab
		}
	}

	r := gitlabSourceConfig.Domains.Rollout

	enabled, err := rollout.Rollout(domain, r.Percentage, r.Stickiness)
	if err != nil {
		log.WithError(err).Error("Rollout error")
		return d.disk
	}

	if enabled {
		return d.gitlab
	}

	return d.disk
}

// IsServerlessDomain checks if a domain requested is a serverless domain we
// need to handle differently.
//
// Domain is a serverless domain when it matches `serverlessDomainRegex`. The
// regular expression is also defined on the gitlab-rails side, see
// https://gitlab.com/gitlab-org/gitlab/-/blob/master/app/models/serverless/domain.rb#L7
func IsServerlessDomain(domain string) bool {
	return serverlessDomainRegex.MatchString(domain)
}