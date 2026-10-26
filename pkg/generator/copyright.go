package generator

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// copyrightEnricher resolves copyright notices from local filesystem caches
// for Go, npm, and Python packages.
type copyrightEnricher struct {
	gomodcache         string
	nodeModulesDir     string
	pythonSitePackages string
}

func newCopyrightEnricher(cfg Config) copyrightEnricher {
	return copyrightEnricher{
		gomodcache:         goModCache(),
		nodeModulesDir:     resolveNodeModulesDir(cfg.NodeModulesDir),
		pythonSitePackages: cfg.PythonSitePackagesDir,
	}
}

func (e copyrightEnricher) enrich(purl, existing string) string {
	if existing != "" {
		return existing
	}

	switch {
	case strings.HasPrefix(purl, "pkg:golang/"):
		return extractGoCopyrightFromCache(e.gomodcache, purl)
	case strings.HasPrefix(purl, "pkg:npm/"):
		return extractNpmCopyright(e.nodeModulesDir, purl)
	case strings.HasPrefix(purl, "pkg:pypi/"):
		return extractPythonCopyright(e.pythonSitePackages, purl)
	}

	return ""
}

// ─── Go ──────────────────────────────────────────────────────────────────────

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
func extractGoCopyrightFromCache(gomodcache, purl string) string {
	// The PURL subpath comes after the qualifiers, so it is cut first.
	// Some SBOM generators (e.g. Aikido) put the module path suffix there:
	// "pkg:golang/github.com/vulcand/oxy@v2.0.3#v2" is the module "github.com/vulcand/oxy/v2".
	purl, subpath, _ := strings.Cut(purl, "#")

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

	// PURL components are percent-encoded (syft emits "+incompatible" as
	// "%2Bincompatible"); decode before mapping them onto the cache layout.
	modulePath, version := pathUnescape(rest[:idx]), pathUnescape(rest[idx+1:])
	if subpath = strings.Trim(pathUnescape(subpath), "/"); subpath != "" {
		modulePath += "/" + subpath
	}

	if !safeRelPath(modulePath) || !safeRelPath(version) {
		return ""
	}

	licensePath := filepath.Join(gomodcache, escapeModulePath(modulePath)+"@"+escapeModulePath(version), "LICENSE")

	return firstCopyrightLine(readFileText(licensePath))
}

// escapeModulePath escapes a module path or version for the module cache
// filesystem layout: each uppercase letter becomes "!" plus its lowercase.
func escapeModulePath(s string) string {
	var b strings.Builder

	for _, r := range s {
		if unicode.IsUpper(r) {
			b.WriteByte('!')
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// ─── npm ─────────────────────────────────────────────────────────────────────

// resolveNodeModulesDir returns the node_modules directory to use.
// If configured is empty, it probes common locations.
func resolveNodeModulesDir(configured string) string {
	if configured != "" {
		return configured
	}

	for _, candidate := range []string{"node_modules", "webui/node_modules"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// extractNpmCopyright reads the copyright notice for an npm package from the
// node_modules directory. It tries LICENSE files first, then falls back to the
// author field in package.json.
func extractNpmCopyright(nodeModulesDir, purl string) string {
	if nodeModulesDir == "" {
		return ""
	}

	name, _ := parseNpmPURL(purl)
	if !safeRelPath(name) {
		return ""
	}

	pkgDir := filepath.Join(nodeModulesDir, filepath.FromSlash(name))

	//nolint:misspell // support British spelling
	licenseFileNames := []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "LICENCE", "LICENCE.md"}

	if filename := findCaseInsensitiveFile(pkgDir, licenseFileNames); filename != "" {
		if c := firstCopyrightLine(readFileText(filename)); c != "" {
			return c
		}
	}

	return npmAuthorCopyright(filepath.Join(pkgDir, "package.json"))
}

func parseNpmPURL(purl string) (string, string) {
	if idx := strings.Index(purl, "?"); idx != -1 {
		purl = purl[:idx]
	}

	rest, ok := strings.CutPrefix(purl, "pkg:npm/")
	if !ok {
		return "", ""
	}

	// Use LastIndex so the "@" in scoped package names (e.g. "@babel/core") is
	// not mistaken for the version separator.
	idx := strings.LastIndex(rest, "@")
	if idx == -1 {
		return "", ""
	}

	// URL-decode for scoped packages encoded as %40babel%2Fcore.
	return pathUnescape(rest[:idx]), pathUnescape(rest[idx+1:])
}

func npmAuthorCopyright(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var pkg struct {
		Author json.RawMessage `json:"author"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Author == nil {
		return ""
	}

	// String form: "Name <email> (url)"
	var s string
	if err := json.Unmarshal(pkg.Author, &s); err == nil {
		return formatAuthorCopyright(s)
	}

	// Object form: {"name": "Name", ...}
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(pkg.Author, &obj); err == nil && obj.Name != "" {
		return "Copyright (c) " + obj.Name
	}

	return ""
}

// formatAuthorCopyright strips the email and URL fragments from an npm author
// string and returns a "Copyright (c) <name>" string.
func formatAuthorCopyright(author string) string {
	if i := strings.Index(author, " <"); i != -1 {
		author = author[:i]
	}

	if i := strings.Index(author, " ("); i != -1 {
		author = author[:i]
	}

	author = strings.TrimSpace(author)
	if author == "" {
		return ""
	}

	return "Copyright (c) " + author
}

// ─── Python ──────────────────────────────────────────────────────────────────

// extractPythonCopyright reads the Author field from the dist-info METADATA
// file for the given PyPI PURL.
func extractPythonCopyright(sitePackagesDir, purl string) string {
	if sitePackagesDir == "" {
		return ""
	}

	if idx := strings.Index(purl, "?"); idx != -1 {
		purl = purl[:idx]
	}

	rest, ok := strings.CutPrefix(purl, "pkg:pypi/")
	if !ok {
		return ""
	}

	idx := strings.LastIndex(rest, "@")
	if idx == -1 {
		return ""
	}

	packageName, version := pathUnescape(rest[:idx]), pathUnescape(rest[idx+1:])
	if !safeRelPath(packageName) || !safeRelPath(version) {
		return ""
	}

	// dist-info directories use the package name as-is or with hyphens replaced
	// by underscores depending on the packaging tool that produced them.
	for _, name := range []string{packageName, strings.ReplaceAll(packageName, "-", "_")} {
		metadataPath := filepath.Join(sitePackagesDir, name+"-"+version+".dist-info", "METADATA")
		if c := pythonAuthorCopyright(readFileText(metadataPath)); c != "" {
			return c
		}
	}

	return ""
}

func pythonAuthorCopyright(metadata string) string {
	for line := range strings.SplitSeq(metadata, "\n") {
		author, ok := strings.CutPrefix(strings.TrimSpace(line), "Author: ")
		if !ok {
			continue
		}

		author = strings.TrimSpace(author)
		if author != "" && !strings.EqualFold(author, "UNKNOWN") {
			return "Copyright (c) " + author
		}
	}

	return ""
}

// ─── Shared ──────────────────────────────────────────────────────────────────

func pathUnescape(s string) string {
	decoded, err := url.PathUnescape(s)
	if err != nil {
		return s
	}

	return decoded
}

// safeRelPath reports whether s can be joined onto a cache directory without
// escaping it. Callers must decode s first, or an encoded "../" slips through.
func safeRelPath(s string) bool {
	if s == "" || strings.ContainsAny(s, "\x00\\") || strings.HasPrefix(s, "/") {
		return false
	}

	for seg := range strings.SplitSeq(s, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return false
		}
	}

	return true
}

// firstCopyrightLine returns the first line in text that starts with "Copyright"
// (case-insensitive), trimmed of surrounding whitespace.
func firstCopyrightLine(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "copyright") {
			return trimmed
		}
	}

	return ""
}

func readFileText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(data)
}

// findCaseInsensitiveFile returns the path of the first file in dir whose name
// matches one of names case-insensitively. Preference follows the order of
// names: "LICENSE" beats "LICENSE.md" even if the latter appears first in dir.
func findCaseInsensitiveFile(dir string, names []string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, name := range names {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if strings.EqualFold(entry.Name(), name) {
				return filepath.Join(dir, entry.Name())
			}
		}
	}

	return ""
}
