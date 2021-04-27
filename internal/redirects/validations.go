package redirects

import (
	"net/http"
	"net/url"
	"strings"

	netlifyRedirects "github.com/tj/go-redirects"
)

// validateURL runs validations against a rule URL.
// Returns `nil` if the URL is valid.
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

// validateRule runs all validation rules on the provided rule.
// Returns `nil` if the rule is valid.
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

// match returns the first redirect or rewrite rule that matches the requested URL
func (r *Redirects) match(url *url.URL) *netlifyRedirects.Rule {
	for _, rule := range r.rules {
		// TODO: Likely this should include host comparison once we have domain-level redirects
		if normalizePath(rule.From) == normalizePath(url.Path) && validateRule(rule) == nil {
			return &rule
		}
	}

	return nil
}
