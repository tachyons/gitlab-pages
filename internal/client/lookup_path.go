package client

import (
	"strings"
)

type LookupPath struct {
	Prefix string `json:"prefix"`
	Path   string `json:"path"`

	NamespaceProject bool   `json:"namespace_project"`
	HTTPSOnly        bool   `json:"https_only"`
	AccessControl    bool   `json:"access_control"`
	ProjectID        uint64 `json:"id"`
}

func (lp *LookupPath) Tail(path string) string {
	if strings.HasPrefix(path, lp.Prefix) {
		return path[len(lp.Prefix):]
	}

	return ""
}
