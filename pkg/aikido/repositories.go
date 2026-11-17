package aikido

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const repositoriesPerPage = 200

type repository struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

// FetchRepoCode returns the Aikido code for a repository name and branch.
func (c *Client) FetchRepoCode(ctx context.Context, name, branch string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("repository name cannot be empty")
	}

	if strings.TrimSpace(branch) == "" {
		return "", fmt.Errorf("branch cannot be empty")
	}

	token, err := c.getToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return c.findRepositoryCode(ctx, token, name, branch)
}

func (c *Client) findRepositoryCode(ctx context.Context, token, name, branch string) (string, error) {
	var repoCode string

	for page := 0; ; page++ {
		repositories, err := c.listRepositories(ctx, token, name, branch, page)
		if err != nil {
			return "", err
		}

		for _, repo := range repositories {
			if repo.Name != name || repo.Branch != branch {
				continue
			}

			if repoCode != "" {
				return "", fmt.Errorf("multiple Aikido repositories match name %q and branch %q", name, branch)
			}

			repoCode = strconv.FormatInt(repo.ID, 10)
		}

		if len(repositories) < repositoriesPerPage {
			break
		}
	}

	if repoCode == "" {
		return "", fmt.Errorf("no active Aikido repository matches name %q and branch %q", name, branch)
	}

	return repoCode, nil
}

func (c *Client) listRepositories(ctx context.Context, token, name, branch string, page int) ([]repository, error) {
	query := url.Values{
		"filter_name":   {name},
		"filter_branch": {branch},
		"page":          {strconv.Itoa(page)},
		"per_page":      {strconv.Itoa(repositoriesPerPage)},
	}

	data, err := c.get(ctx, token, "/api/public/v1/repositories/code?"+query.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to list Aikido repositories: %w", err)
	}

	var repositories []repository
	if err := json.Unmarshal(data, &repositories); err != nil {
		return nil, fmt.Errorf("failed to decode Aikido repositories: %w", err)
	}

	return repositories, nil
}
