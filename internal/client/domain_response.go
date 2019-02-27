package client

import (
	"errors"
	"strings"
)

// DomainResponse describes a configuration for domain,
// like certificate, but also lookup paths to serve the content
type DomainResponse struct {
	Certificate string `json:"certificate"`
	Key         string `json:"certificate_key"`

	LookupPath []LookupPath `json:"lookup_paths"`
}

// GetPath finds a first matching lookup path that should serve the content
func (d *DomainResponse) GetPath(path string) (*LookupPath, error) {
	for _, lp := range d.LookupPath {
		if strings.HasPrefix(path, lp.Prefix) || path+"/" == lp.Prefix {
			return &lp, nil
		}
	}

	return nil, errors.New("lookup path not found")
}
