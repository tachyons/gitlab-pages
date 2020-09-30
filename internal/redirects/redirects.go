// Package redirects provides functions for parsing and rewriting URLs
// according to Netlify style _redirects syntax
package redirects

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	netlifyRedirects "github.com/tj/go-redirects"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const (
	// ConfigFile is the default name of the file containing the redirect rules.
	// It follows Netlify's syntax but we don't support the special options yet like splats, placeholders, query parameters
	//  - https://docs.netlify.com/routing/redirects/
	//  - https://docs.netlify.com/routing/redirects/redirect-options/
	ConfigFile = "_redirects"

	// Check https://gitlab.com/gitlab-org/gitlab-pages/-/issues/472 before increasing this value
	maxConfigSize = 64 * 1024
)

var (
	// ErrNoRedirect is the error thrown when a no redirect rule matches while trying to Rewrite URL.
	// This means that no redirect applies to the URL and you can fallback to serving actual content instead.
	ErrNoRedirect                      = errors.New("no redirect found")
	errConfigNotFound                  = errors.New("_redirects file not found")
	errNeedRegularFile                 = errors.New("_redirects needs to be a regular file (not a directory)")
	errFileTooLarge                    = errors.New("_redirects file too large")
	errFailedToOpenConfig              = errors.New("unable to open _redirects file")
	errFailedToParseConfig             = errors.New("failed to parse _redirects file")
	errFailedToParseURL                = errors.New("unable to parse URL")
	errNoDomainLevelRedirects          = errors.New("no domain-level redirects to outside sites")
	errNoStartingForwardSlashInURLPath = errors.New("url path must start with forward slash /")
	errNoSplats                        = errors.New("splats are not supported")
	errNoPlaceholders                  = errors.New("placeholders are not supported")
	errNoParams                        = errors.New("params not supported")
	errUnsupportedStatus               = errors.New("status not supported")
	errNoForce                         = errors.New("force! not supported")
	regexpPlaceholder                  = regexp.MustCompile(`(?i)/:[a-z]+`)
)

type Redirects struct {
	rules []netlifyRedirects.Rule
	error error
}

// Status maps over each redirect rule and returns any error message
func (r *Redirects) Status() string {
	if r.error != nil {
		return fmt.Sprintf("parse error: %s", r.error.Error())
	}

	messages := make([]string, 0, len(r.rules)+1)
	messages = append(messages, fmt.Sprintf("%d rules", len(r.rules)))

	for i, rule := range r.rules {
		if err := validateRule(rule); err != nil {
			messages = append(messages, fmt.Sprintf("rule %d: error: %s", i+1, err.Error()))
		} else {
			messages = append(messages, fmt.Sprintf("rule %d: valid", i+1))
		}
	}

	return strings.Join(messages, "\n")
}

func validateURL(urlText string) error {
	url, err := url.Parse(urlText)
	if err != nil {
		return errFailedToParseURL
	}

	// No support for domain-level redirects to outside sites:
	// - `https://google.com`
	// - `//google.com`
	if url.Host != "" || url.Scheme != "" {
		return errNoDomainLevelRedirects
	}

	// No parent traversing relative URL's with `./` or `../`
	// No ambiguous URLs like bare domains `GitLab.com`
	if !strings.HasPrefix(url.Path, "/") {
		return errNoStartingForwardSlashInURLPath
	}

	// No support for splats, https://docs.netlify.com/routing/redirects/redirect-options/#splats
	if strings.Contains(url.Path, "*") {
		return errNoSplats
	}

	// No support for placeholders, https://docs.netlify.com/routing/redirects/redirect-options/#placeholders
	if regexpPlaceholder.MatchString(url.Path) {
		return errNoPlaceholders
	}

	return nil
}

func validateRule(r netlifyRedirects.Rule) error {
	if err := validateURL(r.From); err != nil {
		return err
	}

	if err := validateURL(r.To); err != nil {
		return err
	}

	// No support for query parameters, https://docs.netlify.com/routing/redirects/redirect-options/#query-parameters
	if r.Params != nil {
		return errNoParams
	}

	// We strictly validate return status codes
	switch r.Status {
	case http.StatusMovedPermanently, http.StatusFound:
		// noop
	default:
		return errUnsupportedStatus
	}

	// No support for rules that use ! force
	if r.Force {
		return errNoForce
	}

	return nil
}

func normalizePath(path string) string {
	return strings.TrimSuffix(path, "/") + "/"
}

func (r *Redirects) match(url *url.URL) *netlifyRedirects.Rule {
	for _, rule := range r.rules {
		// TODO: Likely this should include host comparison once we have domain-level redirects
		if normalizePath(rule.From) == normalizePath(url.Path) && validateRule(rule) == nil {
			return &rule
		}
	}

	return nil
}

// Rewrite takes in a URL and uses the parsed Netlify rules to rewrite
// the URL to the new location if it matches any rule
func (r *Redirects) Rewrite(url *url.URL) (*url.URL, int, error) {
	rule := r.match(url)
	if rule == nil {
		return nil, 0, ErrNoRedirect
	}

	newURL, err := url.Parse(rule.To)
	log.WithFields(log.Fields{
		"url":         url,
		"newURL":      newURL,
		"err":         err,
		"rule.From":   rule.From,
		"rule.To":     rule.To,
		"rule.Status": rule.Status,
	}).Debug("Rewrite")
	return newURL, rule.Status, err
}

// ParseRedirects decodes Netlify style redirects from the projects `.../public/_redirects`
// https://docs.netlify.com/routing/redirects/#syntax-for-the-redirects-file
func ParseRedirects(ctx context.Context, root vfs.Root) *Redirects {
	fi, err := root.Lstat(ctx, ConfigFile)
	if err != nil {
		return &Redirects{error: errConfigNotFound}
	}

	if !fi.Mode().IsRegular() {
		return &Redirects{error: errNeedRegularFile}
	}

	if fi.Size() > maxConfigSize {
		return &Redirects{error: errFileTooLarge}
	}

	reader, err := root.Open(ctx, ConfigFile)
	if err != nil {
		return &Redirects{error: errFailedToOpenConfig}
	}
	defer reader.Close()

	redirectRules, err := netlifyRedirects.Parse(reader)
	if err != nil {
		return &Redirects{error: errFailedToParseConfig}
	}

	return &Redirects{rules: redirectRules}
}
