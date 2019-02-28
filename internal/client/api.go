package client

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type LookupConfig struct {
	NamespaceProject bool   `json:"namespace_project"`
	HTTPSOnly        bool   `json:"https_only"`
	AccessControl    bool   `json:"access_control"`
	ProjectID        uint64 `json:"id"`
}

type LookupPath struct {
	LookupConfig

	Prefix string `json:"prefix"`
	Path   string `json:"path"`
}

func (lp *LookupPath) Tail(r *http.Request) string {
	if strings.HasPrefix(r.URL.Path, lp.Prefix) {
		return r.URL.Path[len(lp.Path):]
	}

	return ""
}

type DomainResponse struct {
	Domain      string `json:"domain"`
	Certificate string `json:"certificate"`
	Key         string `json:"certificate_key"`

	LookupPath []LookupPath `json:"lookup_paths"`
}

func (d *DomainResponse) GetPath(r *http.Request) *LookupPath {
	for _, lp := range d.LookupPath {
		if strings.HasPrefix(r.RequestURI, lp.Prefix) {
			return &lp
		}
	}

	return nil
}

func RequestDomain(apiUrl, host string) *DomainResponse {
	var values url.Values
	values.Add("host", host)

	resp, err := http.PostForm(apiUrl+"/pages/domain", values)
	if err != nil {
		// Ignore here
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var domainResponse DomainResponse
	err = json.NewDecoder(resp.Body).Decode(&domainResponse)
	if err != nil {
		// Ignore here
		return nil
	}

	return nil
}
