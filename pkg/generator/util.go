package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func writeText(p, s string) error {
	// #nosec G301 -- output directory should be readable in artifacts
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	// #nosec G306 -- generated notices should be readable
	return os.WriteFile(p, []byte(s), 0o644)
}

func readJSON[T any](path string) (T, error) {
	var zero T
	// #nosec G304 -- only reading from trusted paths
	b, err := os.ReadFile(path)
	if err != nil {
		return zero, err
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func uniqSorted(in []string) []string {
	m := make(map[string]struct{}, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			m[s] = struct{}{}
		}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sanitizeID(s string, maxLen int) string {
	s = regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(s, "-")
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return strings.Trim(s, "-")
}
