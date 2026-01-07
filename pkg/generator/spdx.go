package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func fetchText(ctx context.Context, url string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "oss-attributions-generator")

	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, res.Body)
		return "", fmt.Errorf("http %d for %s", res.StatusCode, url)
	}

	b, err := io.ReadAll(res.Body)
	return string(b), err
}

func loadSpdxNameMap(ctx context.Context, spdxVersion string) (map[string]string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/spdx/license-list-data/%s/json/licenses.json", spdxVersion)
	body, err := fetchText(ctx, url)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Licenses []struct {
			ID   string `json:"licenseId"`
			Name string `json:"name"`
		} `json:"licenses"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}

	out := make(map[string]string, len(payload.Licenses))
	for _, l := range payload.Licenses {
		out[l.ID] = l.Name
	}
	return out, nil
}

func getLicenseText(ctx context.Context, cfg Config, licenseID string) (string, error) {
	cachePath := filepath.Join(cfg.OutLicensesDir, licenseID+".txt")

	// #nosec G304 -- cachePath is constructed from controlled inputs
	if b, err := os.ReadFile(cachePath); err == nil {
		return string(b), nil
	}

	if strings.HasPrefix(licenseID, "LicenseRef-") {
		customPath := filepath.Join(cfg.CustomLicenseDir, licenseID+".txt")
		// #nosec G304 -- customPath is constructed from controlled inputs
		b, err := os.ReadFile(customPath)
		if err != nil {
			return "", fmt.Errorf("unknown license %q: expected custom license text at %s", licenseID, customPath)
		}

		return string(b), nil
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/spdx/license-list-data/%s/text/%s.txt", cfg.SPDXVersion, licenseID)
	txt, err := fetchText(ctx, url)
	if err != nil {
		return "", fmt.Errorf("could not fetch SPDX text for %s from %s: %w", licenseID, url, err)
	}

	if err := writeText(cachePath, txt); err != nil {
		return "", err
	}

	return txt, nil
}
