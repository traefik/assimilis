// Package aikido downloads CycloneDX SBOMs from the Aikido API.
package aikido

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client is an Aikido API client.
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// NewClient creates an Aikido API client.
func NewClient(baseURL, clientID, clientSecret string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   httpClient,
	}
}

// DownloadSBOM authenticates with Aikido and downloads a repository's CycloneDX SBOM.
func (c *Client) DownloadSBOM(ctx context.Context, repoCode string) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	endpoint := fmt.Sprintf(
		"/api/public/v1/repositories/code/%s/licenses/export?format=sbom",
		url.PathEscape(repoCode),
	)

	sbom, err := c.get(ctx, token, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to download SBOM: %w", err)
	}

	return sbom, nil
}

func (c *Client) get(ctx context.Context, token, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if responseErr := checkResponse(resp); responseErr != nil {
		return nil, responseErr
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	body := bytes.NewBufferString(`{"grant_type":"client_credentials"}`)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/oauth/token", body)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkResponse(resp); err != nil {
		return "", err
	}

	var result struct {
		AccessToken string `json:"access_token"` //nolint:tagliatelle // Aikido's OAuth response uses snake_case.
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if strings.TrimSpace(result.AccessToken) == "" {
		return "", fmt.Errorf("token response did not contain an access token")
	}

	return result.AccessToken, nil
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("aikido returned HTTP %d", resp.StatusCode)
	}

	return fmt.Errorf("aikido returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
