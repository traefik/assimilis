package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func writeText(p, s string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory for %q: %w", p, err)
	}

	return os.WriteFile(p, []byte(s), 0o644)
}

func readJSON[T any](path string) (T, error) {
	var zero T

	b, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return zero, fmt.Errorf("failed to unmarshal JSON from file %q: %w", path, err)
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

func sanitizeID(s string) string {
	s = regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(s, "-")

	return strings.Trim(s, "-")
}
