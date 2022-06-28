package redirects

import (
	"fmt"
	"regexp"
	"strings"

	netlifyRedirects "github.com/tj/go-redirects"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
)

var (
	regexMultipleSlashes     = regexp.MustCompile(`//+`)
	regexPlaceholderOrSplats = regexp.MustCompile(`(?i)\*|:[a-z]+`)
)

// matchesRule returns `true` if the rule's "from" pattern matches the requested URL.
//
// For example, given a "from" URL like this:
//
//    /a/*/url/with/:placeholders
//
// this function would match URLs like this:
//
//    /a/nice/url/with/text
//    /a/super/extra/nice/url/with/matches
//
// If the first return value is `true`, the second return value is the path that this
// rule should redirect/rewrite to. This path is effectively the rule's "to" path that
// has been templated with all the placeholders (if any) from the originally requested URL.
//
// TODO: Likely these should include host comparison once we have domain-level redirects
// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/601
func matchesRule(rule *netlifyRedirects.Rule, path string) (bool, string) {
	// If the requested URL exactly matches this rule's "from" path,
	// exit early and return the rule's "to" path to avoid building
	// and compiling the regex below.
	// However, only do this if there's nothing to template in the "to" path,
	// to avoid redirect/rewriting to a url with a literal `:placeholder` in it.
	if normalizePath(rule.From) == normalizePath(path) && !regexPlaceholderOrSplats.MatchString(rule.To) {
		return true, rule.To
	}

	// Any logic beyond this point handles placeholders and splats.
	// If the FF_ENABLE_PLACEHOLDERS feature flag isn't enabled, exit now.
	if !feature.RedirectsPlaceholders.Enabled() {
		return false, ""
	}

	var regexSegments []string
	for _, segment := range strings.Split(rule.From, "/") {
		if segment == "" {
			continue
		} else if regexSplat.MatchString(segment) {
			regexSegments = append(regexSegments, `/*(?P<splat>.*)/*`)
		} else if regexPlaceholder.MatchString(segment) {
			segmentName := strings.Replace(segment, ":", "", 1)
			regexSegments = append(regexSegments, fmt.Sprintf(`/+(?P<%s>[^/]+)`, segmentName))
		} else {
			regexSegments = append(regexSegments, "/+"+regexp.QuoteMeta(segment))
		}
	}

	fromRegexString := `(?i)^` + strings.Join(regexSegments, "") + `/*$`
	fromRegex, err := regexp.Compile(fromRegexString)
	if err != nil {
		log.WithFields(log.Fields{
			"fromRegexString": fromRegexString,
			"rule.From":       rule.From,
			"rule.To":         rule.To,
			"rule.Status":     rule.Status,
			"path":            path,
		}).WithError(err).Warnf("matchesRule generated an invalid regex: %q", fromRegexString)

		return false, ""
	}

	template := regexPlaceholderReplacement.ReplaceAllString(rule.To, `${$placeholder}`)
	submatchIndex := fromRegex.FindStringSubmatchIndex(path)

	if submatchIndex == nil {
		return false, ""
	}

	templatedToPath := []byte{}
	templatedToPath = fromRegex.ExpandString(templatedToPath, template, path, submatchIndex)

	// Some replacements result in subsequent slashes. For example, a rule with a "to"
	// like `foo/:splat/bar` will result in a path like `foo//bar` if the splat
	// character matches nothing. To avoid this, replace all instances
	// of multiple subsequent forward slashes with a single forward slash.
	templatedToPath = regexMultipleSlashes.ReplaceAll(templatedToPath, []byte("/"))

	return true, string(templatedToPath)
}

// `match` returns:
// 1. The first valid redirect or rewrite rule that matches the requested URL
// 2. The URL to redirect/rewrite to
//
// If no rule matches, this function returns `nil` and an empty string
func (r *Redirects) match(path string) (*netlifyRedirects.Rule, string) {
	for i := range r.rules {
		if i >= cfg.MaxRuleCount {
			// do not process any more rules
			return nil, ""
		}

		// assign rule to a new var to prevent the following gosec error
		// G601: Implicit memory aliasing in for loop
		rule := r.rules[i]

		if validateRule(rule) != nil {
			continue
		}

		if isMatch, path := matchesRule(&rule, path); isMatch {
			return &rule, path
		}
	}

	return nil, ""
}
