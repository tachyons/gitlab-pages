// Package redirects provides functions for parsing and rewriting URLs
// according to Netlify style _redirects syntax
package redirects

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	netlifyRedirects "github.com/tj/go-redirects"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

const (
	// ConfigFile is the default name of the file containing the redirect rules.
	// It follows Netlify's syntax but we don't support all of the special options yet
	//  - https://docs.netlify.com/routing/redirects/
	//  - https://docs.netlify.com/routing/redirects/redirect-options/
	ConfigFile = "_redirects"

	// Check https://gitlab.com/gitlab-org/gitlab-pages/-/issues/472 before increasing this value
	maxConfigSize = 64 * 1024

	// maxPathSegments is used to limit the number of path segments allowed in rules URLs
	maxPathSegments = 25

	// maxRuleCount is used to limit the total number of rules allowed in _redirects
	maxRuleCount = 1000

	// FFEnablePlaceholders used to check whether placeholder matching is enabled or not
	FFEnablePlaceholders = "FF_ENABLE_PLACEHOLDERS"
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
	errTooManyPathSegments             = fmt.Errorf("url path cannot contain more than %d forward slashes", maxPathSegments)
	errTooManyRules                    = fmt.Errorf("_redirects file may not contain more than %d rules", maxRuleCount)
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

	if err := validateRedirectsFile(r); err != nil {
		messages = append(messages, fmt.Sprintf("error: %s", err.Error()))
	}

	for i, rule := range r.rules {
		if err := validateRule(rule); err != nil {
			messages = append(messages, fmt.Sprintf("rule %d: error: %s", i+1, err.Error()))
		} else {
			messages = append(messages, fmt.Sprintf("rule %d: valid", i+1))
		}
	}

	return strings.Join(messages, "\n")
}

// Rewrite takes in a URL and uses the parsed Netlify rules to rewrite
// the URL to the new location if it matches any rule
func (r *Redirects) Rewrite(originalURL *url.URL) (*url.URL, int, error) {
	rule, newPath := r.match(originalURL.Path)
	if rule == nil {
		return nil, 0, ErrNoRedirect
	}

	newURL, err := url.Parse(newPath)

	log.WithFields(log.Fields{
		"url":         originalURL,
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
