package client

import (
	"errors"
	"strings"
)

type DomainResponse struct {
	Domain      string `json:"domain"`
	Certificate string `json:"certificate"`
	Key         string `json:"certificate_key"`

	LookupPath []LookupPath `json:"lookup_paths"`
}

func (d *DomainResponse) GetPath(path string) (*LookupPath, error) {
	for _, lp := range d.LookupPath {
		if strings.HasPrefix(path, lp.Prefix) || path+"/" == lp.Prefix {
			return &lp, nil
		}
	}

	return nil, errors.New("lookup path not found")
}
