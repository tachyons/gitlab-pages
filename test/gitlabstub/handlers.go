package gitlabstub

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

func defaultAPIHandler(delay time.Duration, pagesRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("host")
		if domain == "127.0.0.1" {
			// shortcut for healthy checkup done by WaitUntilRequestSucceeds
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// to test slow responses from the API
		if delay > 0 {
			time.Sleep(delay)
		}

		// check if predefined response exists
		if responseFn, ok := domainResponses[domain]; ok {
			if err := json.NewEncoder(w).Encode(responseFn(pagesRoot)); err != nil {
				log.Fatal(err)
			}
			return
		}

		// serve lookup from files
		lookupFromFile(domain, w)
	}
}

func defaultAuthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := json.NewEncoder(w).Encode(struct {
			AccessToken string `json:"access_token"`
		}{
			AccessToken: "abc",
		})

		if err != nil {
			log.Fatal(err)
		}
	}
}

func defaultUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer abc" {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func lookupFromFile(domain string, w http.ResponseWriter) {
	fixture, err := os.Open("../../shared/lookups/" + domain + ".json")
	if errors.Is(err, fs.ErrNotExist) {
		w.WriteHeader(http.StatusNoContent)

		log.Printf("GitLab domain %s source stub served 204", domain)
		return
	}

	if err != nil {
		log.Fatal(err)
	}

	defer fixture.Close()

	if _, err = io.Copy(w, fixture); err != nil {
		log.Fatal(err)
	}

	log.Printf("GitLab domain %s source stub served lookup", domain)
}

func handleAccessControlArtifactRequests(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")

	switch {
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/delayed_200.html`).MatchString(r.URL.Path):
		sleepIfAuthorized(authorization, w)
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/404.html`).MatchString(r.URL.Path):
		w.WriteHeader(http.StatusNotFound)
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/500.html`).MatchString(r.URL.Path):
		returnIfAuthorized(authorization, w, http.StatusInternalServerError)
	case regexp.MustCompile(`/api/v4/projects/group/private/jobs/\d+/artifacts/200.html`).MatchString(r.URL.Path):
		returnIfAuthorized(authorization, w, http.StatusOK)
	case regexp.MustCompile(`/api/v4/projects/group/subgroup/private/jobs/\d+/artifacts/200.html`).MatchString(r.URL.Path):
		returnIfAuthorized(authorization, w, http.StatusOK)
	default:
		log.Printf("Unexpected r.URL.RawPath: %q", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusTeapot)
	}
}

func handleAccessControlRequests(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")

	switch {
	case regexp.MustCompile(`/api/v4/projects/1\d{3}/pages_access`).MatchString(r.URL.Path):
		returnIfAuthorized(authorization, w, http.StatusOK)
	case regexp.MustCompile(`/api/v4/projects/2\d{3}/pages_access`).MatchString(r.URL.Path):
		returnIfAuthorized(authorization, w, http.StatusUnauthorized)
	case regexp.MustCompile(`/api/v4/projects/3\d{3}/pages_access`).MatchString(r.URL.Path):
		returnIfAuthorized(authorization, w, http.StatusUnauthorized)
		fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
	default:
		log.Printf("Unexpected r.URL.RawPath: %q", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusTeapot)
	}
}

func returnIfAuthorized(authorization string, w http.ResponseWriter, status int) {
	if authorization != "" {
		checkAuth(authorization)
		w.WriteHeader(status)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func sleepIfAuthorized(authorization string, w http.ResponseWriter) {
	if authorization != "" {
		checkAuth(authorization)
		time.Sleep(2 * time.Second)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func checkAuth(authorization string) {
	if authorization != "Bearer abc" {
		log.Fatalf("expected bearer abc but go %s", authorization)
	}
}
