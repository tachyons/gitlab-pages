package source

import (
	"errors"
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

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(config Config) (Source, error) {
	switch config.DomainConfigSource() {
	case "gitlab":
		return gitlab.New(config)
	case "auto":
		return newAutoSource(config), nil
	case "disk":
		return newDiskServerlessSource(config), nil
	}

	return nil, fmt.Errorf("invalid option for -domain-config-source: %q", config.DomainConfigSource())
}

type notAvailableSource struct{}

func (s notAvailableSource) GetDomain(domain string) (*domain.Domain, error) {
	return nil, errors.New("Source is not available")
}

func (s notAvailableSource) IsReady() bool {
	return false
}

func (s notAvailableSource) Read(string) {
}

type autoSource struct {
	gitlab Source
	disk   Source
}

func (s *autoSource) source(domain string) Source {
	if IsServerlessDomain(domain) {
		return s.gitlab
	}

	if s.gitlab.IsReady() {
		return s.gitlab
	}

	return s.disk
}

func (s *autoSource) GetDomain(domain string) (*domain.Domain, error) {
	return s.source(domain).GetDomain(domain)
}

func (s *autoSource) IsReady() bool {
	return s.gitlab.IsReady() || s.disk.IsReady()
}

func (s *autoSource) Read(rootDomain string) {
	s.disk.Read(rootDomain)
}

func newAutoSource(config Config) *autoSource {
	source := &autoSource{disk: disk.New()}

	glClient, err := gitlab.New(config)
	if err != nil {
		log.WithError(err).Warn("failed to initialize GitLab client for `-domain-config-source=auto`")

		source.gitlab = notAvailableSource{}
	} else {
		source.gitlab = glClient
	}

	return source
}

type diskServerlessSource struct {
	serverless Source
	disk       Source
}

func (s *diskServerlessSource) source(domain string) Source {
	if IsServerlessDomain(domain) {
		return s.serverless
	}

	return s.disk
}

func (s *diskServerlessSource) GetDomain(domain string) (*domain.Domain, error) {
	return s.source(domain).GetDomain(domain)
}

func (s *diskServerlessSource) IsReady() bool {
	return s.disk.IsReady()
}

func (s *diskServerlessSource) Read(rootDomain string) {
	s.disk.Read(rootDomain)
}

func newDiskServerlessSource(config Config) *diskServerlessSource {
	source := &diskServerlessSource{disk: disk.New()}

	glClient, err := gitlab.New(config)
	if err != nil {
		log.WithError(err).Warn("failed to initialize GitLab client for serverless domains")

		source.serverless = notAvailableSource{}
	} else {
		source.serverless = glClient
	}

	return source
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
