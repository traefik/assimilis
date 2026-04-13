package generator

import (
	"testing"

	"github.com/aquasecurity/trivy/pkg/licensing/expression"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeLicenseIDs_UseExplicitID(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{License: &struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "MIT", Name: "Ignored name"}}}

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{"MIT"}, ids)
}

func TestNormalizeLicenseIDs_UseExpression(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "Apache 2.0"}}

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{"Apache-2.0"}, ids)
}

func TestNormalizeLicenseIDs_UseLicenseMap(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "Python Software Foundation License"}}
	licenseMap := map[string]string{"Python Software Foundation License": "PSF-2.0"}

	ids := normalizeLicenseIDs(licenses, licenseMap)
	assert.Equal(t, []string{"PSF-2.0"}, ids)
}

func TestNormalizeLicenseIDs_SPDXKnown(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "mit"}}

	known, ok := expression.SPDXLicenseID("mit")
	assert.True(t, ok)

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{known}, ids)
}

func TestNormalizeLicenseIDs_LicenseRefForUnknown(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "Unknown License"}}
	ids := normalizeLicenseIDs(licenses, nil)
	assert.Len(t, ids, 1)
	assert.Contains(t, ids[0], "LicenseRef-")
}

func TestNormalizeLicenseIDs_LicenseRefIDMappedViaLicenseMap(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{License: &struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "LicenseRef-MIT-X11"}}}

	licenseMap := map[string]string{"LicenseRef-MIT-X11": "MIT"}

	ids := normalizeLicenseIDs(licenses, licenseMap)
	assert.Equal(t, []string{"MIT"}, ids)
}

func TestNormalizeLicenseIDs_LicenseRefIDWithoutMapping(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{License: &struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "LicenseRef-Unknown"}}}

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{"LicenseRef-Unknown"}, ids)
}

func TestMatchLicenseOverride_ExactPURL(t *testing.T) {
	t.Parallel()

	overrides := map[string]string{
		"pkg:npm/config-chain@1.1.13": "MIT",
	}

	assert.Equal(t, "MIT", matchLicenseOverride("pkg:npm/config-chain@1.1.13", overrides))
	assert.Empty(t, matchLicenseOverride("pkg:npm/config-chain@2.0.0", overrides))
}

func TestMatchLicenseOverride_PrefixMatch(t *testing.T) {
	t.Parallel()

	overrides := map[string]string{
		"pkg:golang/std": "BSD-3-Clause",
	}

	assert.Equal(t, "BSD-3-Clause", matchLicenseOverride("pkg:golang/std@go1.25.3", overrides))
	assert.Equal(t, "BSD-3-Clause", matchLicenseOverride("pkg:golang/std@go1.24.0", overrides))
	assert.Empty(t, matchLicenseOverride("pkg:golang/github.com/foo/bar@v1.0.0", overrides))
}

func TestMatchLicenseOverride_StripsQualifiers(t *testing.T) {
	t.Parallel()

	overrides := map[string]string{
		"pkg:golang/github.com/ghodss/yaml": "MIT",
	}

	assert.Equal(t, "MIT", matchLicenseOverride("pkg:golang/github.com/ghodss/yaml@v1.0.0?goarch=arm64&goos=darwin&type=module", overrides))
}

func TestMatchLicenseOverride_NilMap(t *testing.T) {
	t.Parallel()

	assert.Empty(t, matchLicenseOverride("pkg:npm/foo@1.0.0", nil))
}

func TestNormalizeLicenseIDs_CompoundExpressionWithParentheses(t *testing.T) {
	t.Parallel()

	// Regression: parenthesised expressions used to be regex-split, leaving a
	// dangling "(" on the first token so nothing matched SPDX and everything
	// fell through to LicenseRef-.
	licenses := []LicenseChoice{{Expression: "(Apache-2.0 AND BSD-3-Clause)"}}

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{"Apache-2.0", "BSD-3-Clause"}, ids)
}

func TestNormalizeLicenseIDs_CompoundExpressionOR(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "(MIT OR CC0-1.0)"}}

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{"CC0-1.0", "MIT"}, ids)
}

func TestNormalizeLicenseIDs_CompoundExpressionWithoutParentheses(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "Apache-2.0 AND BSD-3-Clause"}}

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{"Apache-2.0", "BSD-3-Clause"}, ids)
}

func TestNormalizeLicenseIDs_CompoundExpressionWithLicenseRef(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "LicenseRef-Custom OR MIT"}}

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{"LicenseRef-Custom", "MIT"}, ids)
}

func TestNormalizeLicenseIDs_CompoundExpressionWithLicenseRefMapped(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "LicenseRef-MIT-X11 OR BSD-3-Clause"}}
	licenseMap := map[string]string{"LicenseRef-MIT-X11": "MIT"}

	ids := normalizeLicenseIDs(licenses, licenseMap)
	assert.Equal(t, []string{"BSD-3-Clause", "MIT"}, ids)
}

func TestNormalizeLicenseIDs_DedupeAndSort(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{
		{Expression: "mit"},
		{Expression: "MIT"},
		{License: &struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{ID: "MIT"}},
	}

	known, ok := expression.SPDXLicenseID("MIT")
	assert.True(t, ok)

	ids := normalizeLicenseIDs(licenses, nil)
	assert.Equal(t, []string{known}, ids)
}
