package api

import (
	"encoding/json"
	"io"
	"sort"
)

// Lookup defines an API lookup action with a response that GitLab sends
type Lookup struct {
	Name   string
	Error  error
	Domain *VirtualDomain
}

func (l *Lookup) ParseDomain(r io.Reader) {
	l.Error = json.NewDecoder(r).Decode(&l.Domain)

	if l.Domain != nil {
		sortLookupsByPrefixLengthDesc(l.Domain.LookupPaths)
	}
}

// Ensure lookupPaths are sorted by prefix length to ensure the group level
// domain with prefix "/" is the last one to be checked.
// See https://gitlab.com/gitlab-org/gitlab-pages/-/issues/576
func sortLookupsByPrefixLengthDesc(lookups []LookupPath) {
	sort.SliceStable(lookups, func(i, j int) bool {
		return len(lookups[i].Prefix) > len(lookups[j].Prefix)
	})
}
