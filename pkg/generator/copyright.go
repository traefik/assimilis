package generator

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// goModCache returns the Go module cache directory, respecting GOMODCACHE and
// GOPATH environment variables with the standard fallback to ~/go/pkg/mod.
func goModCache() string {
	if v := os.Getenv("GOMODCACHE"); v != "" {
		return v
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}

	return filepath.Join(gopath, "pkg", "mod")
}

// extractGoCopyrightFromCache looks up the LICENSE file in the Go module cache
// for the given PURL and returns the first copyright line found.
// Returns an empty string if the file is missing or has no copyright line.
func extractGoCopyrightFromCache(gomodcache, purl string) string {
	// Strip qualifiers (everything after "?").
	if idx := strings.Index(purl, "?"); idx != -1 {
		purl = purl[:idx]
	}

	rest, ok := strings.CutPrefix(purl, "pkg:golang/")
	if !ok {
		return ""
	}

	idx := strings.LastIndex(rest, "@")
	if idx == -1 {
		return ""
	}

	modulePath, version := rest[:idx], rest[idx+1:]
	if modulePath == "" || version == "" {
		return ""
	}

	licensePath := filepath.Join(gomodcache, escapeModulePath(modulePath)+"@"+version, "LICENSE")

	data, err := os.ReadFile(licensePath)
	if err != nil {
		return ""
	}

	return firstCopyrightLine(string(data))
}

// escapeModulePath escapes a Go module path for the module cache filesystem
// layout: each uppercase letter is replaced with "!" followed by its lowercase.
func escapeModulePath(path string) string {
	var b strings.Builder

	for _, r := range path {
		if unicode.IsUpper(r) {
			b.WriteByte('!')
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// firstCopyrightLine returns the first line in text that starts with "Copyright"
// (case-insensitive), trimmed of surrounding whitespace.
func firstCopyrightLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "copyright") {
			return trimmed
		}
	}

	return ""
}
