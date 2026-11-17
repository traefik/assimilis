package aikido

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRepoCode(t *testing.T) {
	match := repository{ID: 7, Name: "Traefik / Proxy+test", Branch: "release/v2.11+test"}
	fullPage := make([]repository, 200)
	fullPageWithMatch := make([]repository, 200)
	fullPageWithMatch[0] = match

	pagesAtLimit := make([][]repository, maxRepositoryPages)
	for page := range pagesAtLimit {
		pagesAtLimit[page] = fullPage
	}

	pagesAtLimit[0] = fullPageWithMatch

	testCases := []struct {
		name          string
		pages         [][]repository
		expectedError string
	}{
		{
			name: "exact repository and branch",
			pages: [][]repository{{
				{ID: 1, Name: match.Name + "-other", Branch: match.Branch},
				{ID: 2, Name: match.Name, Branch: "main"},
				{ID: 3, Name: "traefik", Branch: match.Branch},
				{ID: 4, Name: match.Name, Branch: strings.ToUpper(match.Branch)},
				match,
			}},
		},
		{
			name:  "case-insensitive repository name",
			pages: [][]repository{{{ID: 7, Name: strings.ToUpper(match.Name), Branch: match.Branch}}},
		},
		{
			name:          "case-insensitive duplicate",
			pages:         [][]repository{{match, {ID: 8, Name: strings.ToUpper(match.Name), Branch: match.Branch}}},
			expectedError: "multiple Aikido repositories match",
		},
		{name: "no match", pages: [][]repository{{}}, expectedError: "no active Aikido repository matches"},
		{name: "match on a later page", pages: [][]repository{fullPage, {match}}},
		{
			name:          "duplicate on a later page",
			pages:         [][]repository{fullPageWithMatch, {match}},
			expectedError: "multiple Aikido repositories match",
		},
		{
			name:          "page limit rejects an incomplete lookup",
			pages:         pagesAtLimit,
			expectedError: "aikido repository lookup exceeded 100 pages",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var tokenRequests, listRequests int

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				switch req.URL.Path {
				case "/api/oauth/token":
					tokenRequests++
					_, err := rw.Write([]byte(`{"access_token":"token"}`))
					assert.NoError(t, err)

				case "/api/public/v1/repositories/code":
					listRequests++

					assert.Equal(t, http.MethodGet, req.Method)
					assert.Equal(t, "Bearer token", req.Header.Get("Authorization"))
					assert.Equal(t, match.Name, req.URL.Query().Get("filter_name"))
					assert.Equal(t, match.Branch, req.URL.Query().Get("filter_branch"))
					assert.Equal(t, "200", req.URL.Query().Get("per_page"))

					page, err := strconv.Atoi(req.URL.Query().Get("page"))
					if !assert.NoError(t, err) || !assert.Equal(t, listRequests-1, page) || !assert.Less(t, page, len(testCase.pages)) {
						http.Error(rw, "unexpected page", http.StatusBadRequest)

						return
					}

					assert.NoError(t, json.NewEncoder(rw).Encode(testCase.pages[page]))

				default:
					assert.Fail(t, "unexpected request", req.URL.Path)
					http.NotFound(rw, req)
				}
			}))
			t.Cleanup(server.Close)

			client := NewClient(server.URL, "client", "secret", server.Client())

			repoCode, err := client.FetchRepoCode(t.Context(), match.Name, match.Branch)
			if testCase.expectedError != "" {
				require.ErrorContains(t, err, testCase.expectedError)
				assert.Empty(t, repoCode)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "7", repoCode)
			}

			assert.Equal(t, 1, tokenRequests)
			assert.Equal(t, len(testCase.pages), listRequests)
		})
	}
}

func TestFetchRepoCodeListFailure(t *testing.T) {
	testCases := []struct {
		name          string
		status        int
		body          string
		expectedError string
	}{
		{name: "API error", status: http.StatusForbidden, body: "access denied", expectedError: "HTTP 403"},
		{name: "invalid JSON", status: http.StatusOK, body: `{`, expectedError: "failed to decode Aikido repositories"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if req.URL.Path == "/api/oauth/token" {
					_, err := rw.Write([]byte(`{"access_token":"token"}`))
					assert.NoError(t, err)

					return
				}

				assert.Equal(t, "/api/public/v1/repositories/code", req.URL.Path)
				rw.WriteHeader(testCase.status)
				_, err := rw.Write([]byte(testCase.body))
				assert.NoError(t, err)
			}))
			t.Cleanup(server.Close)

			client := NewClient(server.URL, "client", "secret", server.Client())
			_, err := client.FetchRepoCode(t.Context(), "traefik", "v2.11")
			require.ErrorContains(t, err, testCase.expectedError)
		})
	}
}
