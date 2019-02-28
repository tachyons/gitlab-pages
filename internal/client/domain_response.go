package client

import (
	"net/http"
	"strings"
)

type DomainResponse struct {
	Domain      string `json:"domain"`
	Certificate string `json:"certificate"`
	Key         string `json:"certificate_key"`

	LookupPath []LookupPath `json:"lookup_paths"`
}

func (d *DomainResponse) GetPath(r *http.Request) *LookupPath {
	for _, lp := range d.LookupPath {
		if strings.HasPrefix(r.URL.Path, lp.Prefix) {
			return &lp
		}
	}

	return nil
}
