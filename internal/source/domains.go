package source

import (
	"fmt"
	"regexp"

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
	domains := &Domains{
		// TODO: disable domains.disk https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
		disk: disk.New(),
	}

	domains.setConfigSource(config)

	// Creating a glClient will start polling connectivity in the background
	glClient, err := newGitlabClient(config)
	if err != nil && domains.configSource != sourceDisk {
		return nil, err
	}

	// TODO: handle DomainConfigSource == "auto" https://gitlab.com/gitlab-org/gitlab/-/issues/218358
	// attach gitlab by default when source is not disk (auto, gitlab)
	if domains.configSource != sourceDisk {
		domains.gitlab = glClient
	}

	return domains, nil
}

// defaults to disk
func (d *Domains) setConfigSource(config Config) {
	switch config.DomainConfigSource() {
	case "gitlab":
		// TODO:  https://gitlab.com/gitlab-org/gitlab/-/issues/218357
		d.configSource = sourceGitlab
	case "auto":
		// TODO: https://gitlab.com/gitlab-org/gitlab/-/issues/218358
		d.configSource = sourceAuto
	case "disk":
		fallthrough
	default:
		d.configSource = sourceDisk
	}
}

func newGitlabClient(config Config) (*gitlab.Gitlab, error) {
	if len(config.InternalGitLabServerURL()) == 0 || len(config.GitlabAPISecret()) == 0 {
		return nil, fmt.Errorf("missing -internal-gitlab-server and/or -api-secret-key")
	}

	return gitlab.New(config)
}

// GetDomain retrieves a domain information from a source. We are using two
// sources here because it allows us to switch behavior and the domain source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk source.
func (d *Domains) GetDomain(name string) (*domain.Domain, error) {
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

	// This check is only needed until we enable `d.gitlab` source in all
	// environments (including on-premises installations) followed by removal of
	// `d.disk` source. This can be safely removed afterwards.
	if IsServerlessDomain(domain) {
		return d.gitlab
	}

	if d.configSource == sourceDisk {
		return d.disk
	}

	if d.gitlab.IsReady() {
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
