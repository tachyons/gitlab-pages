package client

import (
	"encoding/json"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type LookupPath struct {
	Prefix string `json:"prefix"`
	Path   string `json:"path"`

	NamespaceProject bool   `json:"namespace_project"`
	HTTPSOnly        bool   `json:"https_only"`
	AccessControl    bool   `json:"access_control"`
	ProjectID        uint64 `json:"id"`
}

type DomainResponse struct {
	Domain      string `json:"domain"`
	Certificate string `json:"certificate"`
	Key         string `json:"certificate_key"`

	LookupPath []LookupPath `json:"lookup_paths"`
}

func RequestDomain(apiUrl, host string) *domain.D {
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
