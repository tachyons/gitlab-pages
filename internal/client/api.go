package client

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// RequestDomain requests the configuration of domain from GitLab
// this provides information where to fetch data from in order to serve
// the domain content
func RequestDomain(apiURL, host string) *DomainResponse {
	values := url.Values{
		"host": []string{host},
	}

	resp, err := http.PostForm(apiURL+"/pages/domain", values)
	if err != nil {
		// Ignore error, or print it
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Ignore responses that are not 200
		return nil
	}

	var domainResponse DomainResponse
	err = json.NewDecoder(resp.Body).Decode(&domainResponse)
	if err != nil {
		// Ignore here
		return nil
	}

	return &domainResponse
}
