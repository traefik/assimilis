package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderText_EmbeddedTemplate(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	m := Model{GeneratedAt: "2026-01-01T00:00:00Z"}

	out, err := renderText(cfg, embedded, m)
	require.NoError(t, err)
	assert.Contains(t, out, "# NOTICE")
	assert.Contains(t, out, "Generated at: 2026-01-01T00:00:00Z")
}

func TestRenderHTML_EmbeddedTemplate(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	m := Model{GeneratedAt: "2026-01-01T00:00:00Z"}

	out, err := renderHTML(cfg, embedded, m)
	require.NoError(t, err)
	assert.Contains(t, out, "Third Party Licenses")
	assert.Contains(t, out, "Generated: 2026-01-01T00:00:00Z")
	assert.Contains(t, out, "<!DOCTYPE html>")
}
