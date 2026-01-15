package generator

import (
	"testing"

	spdxexp "github.com/aquasecurity/trivy/pkg/licensing"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeLicenseIDs_UseExplicitID(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{License: &struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "MIT", Name: "Ignored name"}}}

	ids := normalizeLicenseIDs(licenses, nil, map[string]string{"MIT": "MIT License"})
	assert.Equal(t, []string{"MIT"}, ids)
}

func TestNormalizeLicenseIDs_UseExpression(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "Apache 2.0"}}
	spdxNames := map[string]string{"Apache-2.0": "Apache License 2.0"}

	ids := normalizeLicenseIDs(licenses, nil, spdxNames)
	assert.Equal(t, []string{"Apache-2.0"}, ids)
}

func TestNormalizeLicenseIDs_UseLicenseMap(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "Python Software Foundation License"}}
	licenseMap := map[string]string{"Python Software Foundation License": "PSF-2.0"}
	spdxNames := map[string]string{"PSF-2.0": "Python Software Foundation License 2.0 "}

	ids := normalizeLicenseIDs(licenses, licenseMap, spdxNames)
	assert.Equal(t, []string{"PSF-2.0"}, ids)
}

func TestNormalizeLicenseIDs_SPDXKnown(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "mit"}}
	known := spdxexp.Normalize("mit")
	spdxNames := map[string]string{known: "MIT License"}

	ids := normalizeLicenseIDs(licenses, nil, spdxNames)
	assert.Equal(t, []string{known}, ids)
}

func TestNormalizeLicenseIDs_LicenseRefForUnknown(t *testing.T) {
	t.Parallel()

	licenses := []LicenseChoice{{Expression: "Unknown License"}}
	ids := normalizeLicenseIDs(licenses, nil, map[string]string{})
	assert.Len(t, ids, 1)
	assert.Contains(t, ids[0], "LicenseRef-")
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
	known := spdxexp.Normalize("MIT")
	spdxNames := map[string]string{known: "MIT"}

	ids := normalizeLicenseIDs(licenses, nil, spdxNames)
	assert.Equal(t, []string{known}, ids)
}
