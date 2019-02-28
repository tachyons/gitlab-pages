package client

import (
	"encoding/json"
	"net/http"
	"net/url"
)

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
