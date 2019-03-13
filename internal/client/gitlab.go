package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"
)

type gitlabAPI struct {
	server string
	key    []byte
	client *http.Client
}

func (a *gitlabAPI) IsReady() bool {
	return true
}

// RequestDomain requests the configuration of domain from GitLab
// this provides information where to fetch data from in order to serve
// the domain content
func (a *gitlabAPI) RequestDomain(host string) (*DomainResponse, error) {
	values := url.Values{
		"host": []string{host},
	}

	resp, err := http.PostForm(a.server+"/pages/domain", values)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	resp.Header.Set("Authorization", "token "+string(a.key))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response code: %q", resp.StatusCode)
	}

	var domainResponse DomainResponse
	err = json.NewDecoder(resp.Body).Decode(&domainResponse)
	if err != nil {
		// Ignore here
		return nil, err
	}

	return &domainResponse, nil
}

func NewGitLabClient(server string, key []byte, timeoutSeconds int) API {
	return &gitlabAPI{
		server: strings.TrimRight(server, "/"),
		key:    key,
		client: &http.Client{
			Timeout:   time.Second * time.Duration(timeoutSeconds),
			Transport: httptransport.Transport,
		},
	}
}
