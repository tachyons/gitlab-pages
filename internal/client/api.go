package client

import (
	"encoding/json"
	"net/http"
	"net/url"
)

func RequestDomain(apiUrl, host string) *DomainResponse {
	values := url.Values{
		"host": []string{host},
	}

	resp, err := http.PostForm(apiUrl+"/pages/domain", values)
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
