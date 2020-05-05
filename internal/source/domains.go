package source

import (
	"errors"
	"strings"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

var (
	// errSourceNotConfigured will be returned when neither disk nor gitlab sources are configured
	errSourceNotConfigured = errors.New("source not configured")
)

// Domains struct represents a map of all domains supported by pages. It is
// currently using two sources during the transition to the new GitLab domains
// source.
type Domains struct {
	gitlab Source
	disk   *disk.Disk // legacy disk source
}

func useDiskConfigSource(config Config) bool {
	return config.GitlabDisableAPIConfigurationSource() || len(config.InternalGitLabServerURL()) == 0 || len(config.GitlabAPISecret()) == 0
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(config Config) (*Domains, error) {
	// fallback to disk if these values are empty
	if useDiskConfigSource(config) {
		log.Warn("disk source will be deprecated soon https://gitlab.com/gitlab-org/gitlab/-/issues/210010")
		return &Domains{disk: disk.New()}, nil
	}

	gl, err := gitlab.New(config)
	if err != nil {
		if strings.Contains(err.Error(), client.ConnectionErrorMsg) {
			log.WithError(err).Warn("GitLab API is not configured https://gitlab.com/gitlab-org/gitlab/-/issues/210010")
			return &Domains{disk: disk.New()}, nil
		}
		return nil, err
	}

	return &Domains{
		gitlab: gl,
	}, nil
}

// GetDomain retrieves a domain information from a source. We are using two
// sources here because it allows us to switch behavior and the domain source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk source.
func (d *Domains) GetDomain(name string) (*domain.Domain, error) {
	source, err := d.source()
	if err != nil {
		return nil, err
	}
	return source.GetDomain(name)
}

// Read starts the disk domain source. It is DEPRECATED, because we want to
// remove it entirely when disk source gets removed.
func (d *Domains) Read(rootDomain string) {
	if d.disk != nil {
		d.disk.Read(rootDomain)
	}
}

// IsReady checks if the disk domain source managed to traverse entire pages
// filesystem and is ready for use. It is DEPRECATED, because we want to remove
// it entirely when disk source gets removed.
func (d *Domains) IsReady() bool {
	// return true if d.disk is nil while we remove all of the disk source code
	// TODO https://gitlab.com/gitlab-org/gitlab-pages/-/issues/379
	return d.disk == nil || d.disk.IsReady()
}

func (d *Domains) source() (Source, error) {
	if d.gitlab != nil {
		return d.gitlab, nil
	} else if d.disk != nil {
		return d.disk, nil
	}
	return nil, errSourceNotConfigured
}
