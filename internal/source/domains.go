package source

import (
	"fmt"
	"regexp"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
)

var (
	// serverlessDomainRegex is a regular expression we use to check if a domain
	// is a serverless domain, to short circuit gitlab source rollout. It can be
	// removed after the rollout is done
	serverlessDomainRegex = regexp.MustCompile(`^[^.]+-[[:xdigit:]]{2}a1[[:xdigit:]]{10}f2[[:xdigit:]]{2}[[:xdigit:]]+-?.*`)
)

type configSource int

const (
	sourceGitlab configSource = iota
	sourceDisk
	sourceAuto
)

// Domains struct represents a map of all domains supported by pages. It is
// currently using two sources during the transition to the new GitLab domains
// source.
type Domains struct {
	configSource configSource
	gitlab       Source
	disk         *disk.Disk // legacy disk source
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(config Config) (*Domains, error) {
	domains := &Domains{}
	if err := domains.setConfigSource(config); err != nil {
		return nil, err
	}

	return domains, nil
}

// setConfigSource and initialize gitlab source
// returns error if -domain-config-source is not valid
// returns error if -domain-config-source=gitlab and init fails
func (d *Domains) setConfigSource(config Config) error {
	// attach gitlab by default when source is not disk (auto, gitlab)
	switch config.DomainConfigSource() {
	case "gitlab":
		d.configSource = sourceGitlab
		return d.setGitLabClient(config)
	case "auto":
		// TODO: handle DomainConfigSource == "auto" https://gitlab.com/gitlab-org/gitlab/-/issues/218358
		d.configSource = sourceAuto
		// enable disk for auto for now
		d.disk = disk.New(config.RootDomain())
		return d.setGitLabClient(config)
	case "disk":
		// TODO: disable domains.disk https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
		d.configSource = sourceDisk
		d.disk = disk.New(config.RootDomain())
	default:
		return fmt.Errorf("invalid option for -domain-config-source: %q", config.DomainConfigSource())
	}

	return nil
}

// setGitLabClient when domain-config-source is `gitlab` or `auto`, only return error for `gitlab` source
func (d *Domains) setGitLabClient(config Config) error {
	// We want to notify users about any API issues
	// Creating a glClient will start polling connectivity in the background
	// and spam errors in log
	glClient, err := gitlab.New(config)
	if err != nil {
		if d.configSource == sourceGitlab {
			return err
		}

		log.WithError(err).Warn("failed to initialize GitLab client for `-domain-config-source=auto`")

		return nil
	}

	d.gitlab = glClient

	return nil
}

// GetDomain retrieves a domain information from a source. We are using two
// sources here because it allows us to switch behavior and the domain source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk source.
func (d *Domains) GetDomain(name string) (*domain.Domain, error) {
	return d.source(name).GetDomain(name)
}

// IsReady checks if the disk domain source managed to traverse entire pages
// filesystem and is ready for use. It is DEPRECATED, because we want to remove
// it entirely when disk source gets removed.
func (d *Domains) IsReady() bool {
	switch d.configSource {
	case sourceGitlab:
		return d.gitlab.IsReady()
	case sourceDisk:
		return d.disk.IsReady()
	case sourceAuto:
		// TODO: implement auto https://gitlab.com/gitlab-org/gitlab/-/issues/218358, default to disk for now
		return d.disk.IsReady()
	}

	return false
}

func (d *Domains) source(domain string) Source {
	if d.gitlab == nil {
		return d.disk
	}

	// This check is only needed until we enable `d.gitlab` source in all
	// environments (including on-premises installations) followed by removal of
	// `d.disk` source. This can be safely removed afterwards.
	if IsServerlessDomain(domain) {
		return d.gitlab
	}

	if d.configSource == sourceDisk {
		return d.disk
	}

	// TODO: handle sourceAuto https://gitlab.com/gitlab-org/gitlab/-/issues/218358
	// check IsReady for sourceAuto for now
	if d.configSource == sourceGitlab || d.gitlab.IsReady() {
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
