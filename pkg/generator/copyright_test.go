package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Shared helpers ──────────────────────────────────────────────────────────

func TestFirstCopyrightLine(t *testing.T) {
	t.Parallel()

	text := "Mozilla Public License, version 2.0\n\nCopyright (c) 2015 HashiCorp, Inc.\n\nRedistribution..."
	assert.Equal(t, "Copyright (c) 2015 HashiCorp, Inc.", firstCopyrightLine(text))
}

func TestFirstCopyrightLine_CaseInsensitive(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "COPYRIGHT 2009 The Go Authors.", firstCopyrightLine("COPYRIGHT 2009 The Go Authors.\n\nBSD..."))
}

func TestFirstCopyrightLine_NoMatch(t *testing.T) {
	t.Parallel()

	assert.Empty(t, firstCopyrightLine("MIT License\n\nPermission is hereby granted..."))
}

// ─── Go ──────────────────────────────────────────────────────────────────────

func TestEscapeModulePath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "github.com/!burnt!sushi/toml", escapeModulePath("github.com/BurntSushi/toml"))
	assert.Equal(t, "golang.org/x/sync", escapeModulePath("golang.org/x/sync"))
	assert.Equal(t, "github.com/hashicorp/go-retryablehttp", escapeModulePath("github.com/hashicorp/go-retryablehttp"))
}

func TestExtractGoCopyrightFromCache(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modDir := filepath.Join(dir, "golang.org", "x", "sync@v0.19.0")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "LICENSE"), []byte("Copyright 2009 The Go Authors.\n\nBSD-3-Clause..."), 0o644))

	got := extractGoCopyrightFromCache(dir, "pkg:golang/golang.org/x/sync@v0.19.0?goarch=arm64&goos=darwin&type=module")
	assert.Equal(t, "Copyright 2009 The Go Authors.", got)
}

func TestExtractGoCopyrightFromCache_EscapedPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modDir := filepath.Join(dir, "github.com", "!burnt!sushi", "toml@v1.3.2")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "LICENSE"), []byte("The MIT License (MIT)\n\nCopyright (c) 2013 TOML Authors"), 0o644))

	got := extractGoCopyrightFromCache(dir, "pkg:golang/github.com/BurntSushi/toml@v1.3.2")
	assert.Equal(t, "Copyright (c) 2013 TOML Authors", got)
}

func TestExtractGoCopyrightFromCache_NotFound(t *testing.T) {
	t.Parallel()

	assert.Empty(t, extractGoCopyrightFromCache("/nonexistent/path", "pkg:golang/golang.org/x/sync@v0.19.0"))
}

// ─── npm ─────────────────────────────────────────────────────────────────────

func TestParseNpmPURL(t *testing.T) {
	t.Parallel()

	name, ver := parseNpmPURL("pkg:npm/lodash@4.17.21")
	assert.Equal(t, "lodash", name)
	assert.Equal(t, "4.17.21", ver)

	// Scoped package, non-encoded
	name, ver = parseNpmPURL("pkg:npm/@babel/core@7.25.7")
	assert.Equal(t, "@babel/core", name)
	assert.Equal(t, "7.25.7", ver)

	// Scoped package, URL-encoded
	name, ver = parseNpmPURL("pkg:npm/%40babel%2Fcore@7.25.7")
	assert.Equal(t, "@babel/core", name)
	assert.Equal(t, "7.25.7", ver)
}

func TestExtractNpmCopyright_LicenseFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "lodash")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "LICENSE"), []byte("MIT License\n\nCopyright (c) 2012-2018 The Dojo Foundation <http://dojofoundation.org/>"), 0o644))

	got := extractNpmCopyright(dir, "pkg:npm/lodash@4.17.21")
	assert.Equal(t, "Copyright (c) 2012-2018 The Dojo Foundation <http://dojofoundation.org/>", got)
}

func TestExtractNpmCopyright_PackageJSONAuthorString(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "some-pkg")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	pkg := map[string]any{"author": "Jane Doe <jane@example.com> (https://jane.dev)"}

	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("Failed to marshal package.json: %v", err)
	}

	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), data, 0o644))

	got := extractNpmCopyright(dir, "pkg:npm/some-pkg@1.0.0")
	assert.Equal(t, "Copyright (c) Jane Doe", got)
}

func TestExtractNpmCopyright_PackageJSONAuthorObject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "some-pkg")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))

	pkg := map[string]any{"author": map[string]any{"name": "Acme Corp", "email": "hi@acme.com"}}

	data, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("Failed to marshal package.json: %v", err)
	}

	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "package.json"), data, 0o644))

	got := extractNpmCopyright(dir, "pkg:npm/some-pkg@1.0.0")
	assert.Equal(t, "Copyright (c) Acme Corp", got)
}

func TestExtractNpmCopyright_ScopedPackage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "@babel", "core")
	require.NoError(t, os.MkdirAll(pkgDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pkgDir, "LICENSE"), []byte("MIT\n\nCopyright (c) 2014-present Sebastian McKenzie"), 0o644))

	got := extractNpmCopyright(dir, "pkg:npm/@babel/core@7.25.7")
	assert.Equal(t, "Copyright (c) 2014-present Sebastian McKenzie", got)
}

func TestExtractNpmCopyright_EmptyDir(t *testing.T) {
	t.Parallel()

	assert.Empty(t, extractNpmCopyright("", "pkg:npm/lodash@4.17.21"))
}

// ─── Python ──────────────────────────────────────────────────────────────────

func TestExtractPythonCopyright(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	distInfo := filepath.Join(dir, "requests-2.28.0.dist-info")
	require.NoError(t, os.MkdirAll(distInfo, 0o755))

	metadata := "Metadata-Version: 2.1\nName: requests\nVersion: 2.28.0\nAuthor: Kenneth Reitz\nAuthor-email: me@kennethreitz.org\n"
	require.NoError(t, os.WriteFile(filepath.Join(distInfo, "METADATA"), []byte(metadata), 0o644))

	got := extractPythonCopyright(dir, "pkg:pypi/requests@2.28.0")
	assert.Equal(t, "Copyright (c) Kenneth Reitz", got)
}

func TestExtractPythonCopyright_HyphenToUnderscore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Packaging tool stored it with underscores
	distInfo := filepath.Join(dir, "black_formatter-24.1.0.dist-info")
	require.NoError(t, os.MkdirAll(distInfo, 0o755))

	metadata := "Metadata-Version: 2.1\nName: black-formatter\nVersion: 24.1.0\nAuthor: Łukasz Langa\n"
	require.NoError(t, os.WriteFile(filepath.Join(distInfo, "METADATA"), []byte(metadata), 0o644))

	got := extractPythonCopyright(dir, "pkg:pypi/black-formatter@24.1.0")
	assert.Equal(t, "Copyright (c) Łukasz Langa", got)
}

func TestExtractPythonCopyright_UnknownAuthor(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	distInfo := filepath.Join(dir, "pkg-1.0.0.dist-info")
	require.NoError(t, os.MkdirAll(distInfo, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(distInfo, "METADATA"), []byte("Author: UNKNOWN\n"), 0o644))

	assert.Empty(t, extractPythonCopyright(dir, "pkg:pypi/pkg@1.0.0"))
}

func TestExtractPythonCopyright_EmptyDir(t *testing.T) {
	t.Parallel()

	assert.Empty(t, extractPythonCopyright("", "pkg:pypi/requests@2.28.0"))
}
