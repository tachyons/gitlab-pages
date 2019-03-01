package client

import (
	"strings"
)

// LookupPath describes a single mapping between HTTP Prefix
// and actual data on disk
type LookupPath struct {
	Prefix      string `json:"prefix"`
	Path        string `json:"disk_path"`
	ArchivePath string `json:"archive_path"`

	NamespaceProject bool   `json:"namespace_project"`
	HTTPSOnly        bool   `json:"https_only"`
	AccessControl    bool   `json:"access_control"`
	ProjectID        uint64 `json:"id"`
}

// Tail returns a relative path to full path to serve the content
func (lp *LookupPath) Tail(path string) string {
	if strings.HasPrefix(path, lp.Prefix) {
		return path[len(lp.Prefix):]
	}

	return ""
}
