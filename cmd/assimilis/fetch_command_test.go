package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/assimilis/v2/pkg/generator"
	"github.com/urfave/cli/v3"
)

func TestFetchCommandModes(t *testing.T) {
	const exportError = "failed to download SBOM: aikido returned HTTP 404: export unavailable"

	testCases := []struct {
		name           string
		args           []string
		env            map[string]string
		expectedOutput string
		expectedError  string
		lookups        int
		exportCode     string
		exports        int
		noCredentials  bool
	}{
		{
			name:           "repo code from flags",
			args:           []string{"--type", "repo-code", "--repo-name", testRepoName, "--branch", "v2.11"},
			expectedOutput: "7\n",
			lookups:        1,
		},
		{
			name:           "repo code from environment",
			env:            map[string]string{"AIKIDO_FETCH_TYPE": "repo-code", "AIKIDO_REPO_NAME": testRepoName, "AIKIDO_BRANCH": "v2.11"},
			expectedOutput: "7\n",
			lookups:        1,
		},
		{
			name:           "lookup trims repository name and branch",
			args:           []string{"--type", "repo-code", "--aikido-repo-name", " " + testRepoName + " ", "--branch", " v2.11 "},
			expectedOutput: "7\n",
			lookups:        1,
		},
		{
			name:           "repo code override without credentials",
			args:           []string{"--type", "repo-code", "--repo-code", "99"},
			expectedOutput: "99\n",
			noCredentials:  true,
		},
		{
			name:           "padded repository code override",
			args:           []string{"--type", "repo-code", "--repo-code", " 99 "},
			expectedOutput: "99\n",
			noCredentials:  true,
		},
		{
			name:           "whitespace-only Aikido name falls back",
			args:           []string{"--type", "repo-code", "--repo-name", testRepoName, "--branch", "v2.11"},
			env:            map[string]string{"AIKIDO_REPO_NAME": " \t "},
			expectedOutput: "7\n",
			lookups:        1,
		},
		{
			name:          "default SBOM with automatic lookup",
			args:          []string{"--repo-name", testRepoName, "--branch", "v2.11"},
			expectedError: exportError,
			lookups:       1,
			exportCode:    "7",
			exports:       1,
		},
		{
			name:          "Aikido name overrides filename and environment",
			args:          []string{"--repo-name", "output-name", "--aikido-repo-name", testRepoName, "--branch", "v2.11"},
			env:           map[string]string{"AIKIDO_REPO_NAME": "other-name"},
			expectedError: exportError,
			lookups:       1,
			exportCode:    "7",
			exports:       1,
		},
		{
			name:          "explicit code overrides lookup and environment",
			args:          []string{"--repo-name", testRepoName, "--type", "sbom", "--repo-code", "99", "--aikido-repo-name", "other-name"},
			env:           map[string]string{"AIKIDO_REPO_CODE": "88", "AIKIDO_BRANCH": "v2.11"},
			expectedError: exportError,
			exportCode:    "99",
			exports:       1,
		},
		{
			name:          "legacy code from environment",
			args:          []string{"--repo-name", testRepoName},
			env:           map[string]string{"AIKIDO_REPO_CODE": "88"},
			expectedError: exportError,
			exportCode:    "88",
			exports:       1,
		},
		{name: "invalid type", args: []string{"--type", "invalid"}, expectedError: `unsupported fetch type "invalid": expected sbom or repo-code`},
		{name: "missing repository name", args: []string{"--type", "repo-code"}, expectedError: "repository name cannot be empty"},
		{name: "missing branch", args: []string{"--type", "repo-code", "--repo-name", testRepoName}, expectedError: "branch cannot be empty"},
		{name: "missing SBOM repo name", args: []string{"--repo-code", "99"}, expectedError: "--repo-name cannot be empty"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("AIK_CLIENT", "client")
			t.Setenv("AIK_SECRET", "secret")
			t.Setenv("AIKIDO_FETCH_TYPE", "sbom")
			t.Setenv("AIKIDO_REPO_CODE", "")
			t.Setenv("AIKIDO_REPO_NAME", "")
			t.Setenv("AIKIDO_BRANCH", "")

			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			if testCase.noCredentials {
				t.Setenv("AIK_CLIENT", "")
				t.Setenv("AIK_SECRET", "")
			}

			var requests, lookups, exports int

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				requests++

				switch {
				case req.URL.Path == "/api/oauth/token":
					_, err := rw.Write([]byte(`{"access_token":"token"}`))
					assert.NoError(t, err)

				case req.URL.Path == "/api/public/v1/repositories/code":
					lookups++

					assert.Equal(t, "v2.11", req.URL.Query().Get("filter_branch"))
					assert.Equal(t, testRepoName, req.URL.Query().Get("filter_name"))

					_, err := rw.Write([]byte(`[{"id":7,"name":"repo","branch":"v2.11"}]`))
					assert.NoError(t, err)

				case strings.HasSuffix(req.URL.Path, "/licenses/export"):
					exports++

					assert.Equal(t, "/api/public/v1/repositories/code/"+testCase.exportCode+"/licenses/export", req.URL.Path)
					// Stop before generation so this test only exercises command selection and lookup.
					http.Error(rw, "export unavailable", http.StatusNotFound)

				default:
					assert.Fail(t, "unexpected request", req.URL.Path)
					http.NotFound(rw, req)
				}
			}))
			t.Cleanup(server.Close)
			t.Setenv("AIKIDO_BASE_URL", server.URL)

			var output bytes.Buffer

			cfg := generator.DefaultConfig()
			command := &cli.Command{
				Name:     "assimilis",
				Flags:    buildFlags(&cfg),
				Commands: []*cli.Command{buildFetchCommand(&cfg)},
				Writer:   &output,
			}

			args := append([]string{"assimilis", "fetch"}, testCase.args...)

			err := command.Run(t.Context(), args)
			if testCase.expectedError != "" {
				require.EqualError(t, err, testCase.expectedError)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.expectedOutput, output.String())
			assert.Equal(t, testCase.lookups, lookups)
			assert.Equal(t, testCase.exports, exports)

			if testCase.noCredentials {
				assert.Zero(t, requests)
			}
		})
	}
}
