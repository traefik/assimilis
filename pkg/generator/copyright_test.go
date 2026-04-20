package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscapeModulePath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "github.com/!burnt!sushi/toml", escapeModulePath("github.com/BurntSushi/toml"))
	assert.Equal(t, "golang.org/x/sync", escapeModulePath("golang.org/x/sync"))
	assert.Equal(t, "github.com/hashicorp/go-retryablehttp", escapeModulePath("github.com/hashicorp/go-retryablehttp"))
}

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

func TestExtractGoCopyrightFromCache_NonGolangPURL(t *testing.T) {
	t.Parallel()

	assert.Empty(t, extractGoCopyrightFromCache(t.TempDir(), "pkg:npm/lodash@4.17.21"))
}
