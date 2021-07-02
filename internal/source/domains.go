package source

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
)

type configSource int

const (
	sourceGitlab configSource = iota
	// Disk source is deprecated and support will be removed in 14.3
	// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
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
func NewDomains(source string, cfg *config.GitLab) (*Domains, error) {
	domains := &Domains{}
	if err := domains.setConfigSource(source, cfg); err != nil {
		return nil, err
	}

	return domains, nil
}

// setConfigSource and initialize gitlab source
// returns error if -domain-config-source is not valid
// returns error if -domain-config-source=gitlab and init fails
func (d *Domains) setConfigSource(source string, cfg *config.GitLab) error {
	switch source {
	case "gitlab":
		d.configSource = sourceGitlab
		return d.setGitLabClient(cfg)
	case "auto":
		d.configSource = sourceAuto
		// enable disk for auto for now
		d.disk = disk.New()
		return d.setGitLabClient(cfg)
	case "disk":
		// TODO: disable domains.disk https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
		d.configSource = sourceDisk
		d.disk = disk.New()
	default:
		return fmt.Errorf("invalid option for -domain-config-source: %q", source)
	}

	return nil
}

// setGitLabClient when domain-config-source is `gitlab` or `auto`, only return error for `gitlab` source
func (d *Domains) setGitLabClient(cfg *config.GitLab) error {
	// We want to notify users about any API issues
	// Creating a glClient will start polling connectivity in the background
	// and spam errors in log
	glClient, err := gitlab.New(cfg)
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
func (d *Domains) GetDomain(ctx context.Context, name string) (*domain.Domain, error) {
	return d.source(name).GetDomain(ctx, name)
}

// Read starts the disk domain source. It is DEPRECATED, because we want to
// remove it entirely when disk source gets removed.
func (d *Domains) Read(rootDomain string) {
	// start disk.Read for sourceDisk and sourceAuto
	if d.configSource != sourceGitlab {
		d.disk.Read(rootDomain)
	}
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
		// if gitlab is configured and is ready
		if d.gitlab != nil && d.gitlab.IsReady() {
			return true
		}

		return d.disk.IsReady()
	default:
		return false
	}
}

func (d *Domains) source(domain string) Source {
	switch d.configSource {
	case sourceDisk:
		return d.disk
	case sourceGitlab:
		return d.gitlab
	default:
		if d.gitlab != nil && d.gitlab.IsReady() {
			return d.gitlab
		}

		return d.disk
	}
}
