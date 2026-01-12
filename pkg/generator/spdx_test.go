package generator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchText_OK(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	}))
	defer srv.Close()

	txt, err := fetchText(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "test", txt)
}

func TestFetchText_Fail(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 not found"))
	}))
	defer srv.Close()

	_, err := fetchText(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http 404")
}

func TestGetLicenseText_ReturnCachedFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg := Config{OutLicensesDir: tmp, SPDXVersion: "v0"}
	licenseID := "MIT"
	cachePath := filepath.Join(tmp, licenseID+".txt")
	require.NoError(t, os.WriteFile(cachePath, []byte("cached"), 0o644))

	txt, err := getLicenseText(context.Background(), cfg, licenseID)
	require.NoError(t, err)
	assert.Equal(t, "cached", txt)
}

func TestGetLicenseText_LicenseRefReadCustomText(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg := Config{OutLicensesDir: tmp}
	licenseID := "LicenseRef-Custom-Text"

	customPath := filepath.Join(tmp, "custom", licenseID+".txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(customPath), 0o755))
	require.NoError(t, os.WriteFile(customPath, []byte("custom text"), 0o644))

	txt, err := getLicenseText(context.Background(), cfg, licenseID)
	require.NoError(t, err)
	assert.Equal(t, "custom text", txt)
}

func TestGetLicenseText_LicenseRefMissingCustomText(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfg := Config{OutLicensesDir: tmp}
	licenseID := "LicenseRef-Missing"

	_, err := getLicenseText(context.Background(), cfg, licenseID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected custom license text")
	assert.Contains(t, err.Error(), filepath.Join(tmp, "custom", licenseID+".txt"))
}
