package generator

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnknownLicensesError_IncludesIDs(t *testing.T) {
	t.Parallel()

	err := UnknownLicensesError{
		IDs:              []string{"LicenseRef-MIT-X11", "LicenseRef-Unknown"},
		CustomLicenseDir: "third_party/licenses/custom",
	}
	msg := err.Error()

	require.Contains(t, err.IDs, "LicenseRef-MIT-X11")
	require.Contains(t, err.IDs, "LicenseRef-Unknown")
	require.Contains(t, msg, "third_party/licenses/custom")
}

func TestBuildOverview_SortByCountAndID(t *testing.T) {
	t.Parallel()

	licenses := []LicenseBlock{
		{ID: "MIT", Name: "foo", UsedBy: []OutComponent{{Name: "foo"}}},
		{ID: "Apache-2.0", Name: "bar", UsedBy: []OutComponent{{Name: "bar"}, {Name: "baz"}}},
		{ID: "ISC", Name: "qux", UsedBy: []OutComponent{{Name: "qux"}}},
	}
	overview := buildOverview(licenses)

	require.Equal(t, "Apache-2.0", overview[0].ID)
	require.Equal(t, []string{"ISC", "MIT"}, []string{overview[1].ID, overview[2].ID})
}

func TestBuildNotices_FilterEmptyCopyright(t *testing.T) {
	t.Parallel()

	in := map[string]OutComponent{
		"1": {Name: "foo", Version: "1", Copyright: "c foo"},
		"2": {Name: "bar", Version: "2", Copyright: "c bar"},
		"3": {Name: "bar", Version: "1", Copyright: ""},
	}

	out := buildNotices(in)

	require.Len(t, out, 2)
	require.Equal(t, "bar", out[0].Name)
	require.Equal(t, "2", out[0].Version)
	require.Equal(t, "foo", out[1].Name)
}

func TestBuildIndex_LicenseOverrideForComponentWithoutLicense(t *testing.T) {
	t.Parallel()

	components := []Component{
		{Name: "std", Version: "go1.25.3", PURL: "pkg:golang/std@go1.25.3"},
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Licenses: []LicenseChoice{
			{License: &struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "MIT"}},
		}},
	}
	overrides := map[string]string{
		"pkg:golang/std": "BSD-3-Clause",
	}

	byLicense, byKey, missing := buildIndex(components, Filters{}, nil, overrides, copyrightEnricher{})

	require.Contains(t, byLicense, "BSD-3-Clause")
	require.Contains(t, byLicense, "MIT")
	require.Empty(t, missing)
	require.Equal(t, []string{"BSD-3-Clause"}, byKey["pkg:golang/std@go1.25.3"].LicenseIDs)
}

func TestBuildIndex_OverrideReplacesExistingLicense(t *testing.T) {
	t.Parallel()

	components := []Component{
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Licenses: []LicenseChoice{
			{License: &struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "Apache-2.0"}},
		}},
	}
	overrides := map[string]string{
		"pkg:npm/foo": "MIT",
	}

	_, byKey, missing := buildIndex(components, Filters{}, nil, overrides, copyrightEnricher{})

	// missing-licenses entries take priority and correct wrong licenses from the SBOM.
	require.Equal(t, []string{"MIT"}, byKey["pkg:npm/foo@1.0.0"].LicenseIDs)
	require.Empty(t, missing)
}

func TestBuildIndex_CompoundLicenseOverride(t *testing.T) {
	t.Parallel()

	components := []Component{
		{
			Name:    "foo",
			Version: "1.0.0",
			PURL:    "pkg:npm/foo@1.0.0",
			Licenses: []LicenseChoice{
				{License: &struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{ID: "MIT"}},
			},
		},
	}
	overrides := map[string]string{
		"pkg:npm/foo": "Apache-2.0 AND CC-BY-SA-4.0",
	}

	byLicense, byKey, missing := buildIndex(
		components,
		Filters{},
		nil,
		overrides,
		copyrightEnricher{},
	)

	require.Contains(t, byLicense, "Apache-2.0")
	require.Contains(t, byLicense, "CC-BY-SA-4.0")
	require.Equal(
		t,
		[]string{"Apache-2.0", "CC-BY-SA-4.0"},
		byKey["pkg:npm/foo@1.0.0"].LicenseIDs,
	)
	require.Empty(t, missing)
}

func TestBuildIndex_MergesDuplicateComponents(t *testing.T) {
	t.Parallel()

	components := []Component{
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Licenses: []LicenseChoice{
			{License: &struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "MIT"}},
		}},
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Copyright: "(c) Foo Inc", Licenses: []LicenseChoice{
			{License: &struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "Apache-2.0"}},
		}},
	}

	_, byKey, missing := buildIndex(components, Filters{}, nil, nil, copyrightEnricher{})

	merged := byKey["pkg:npm/foo@1.0.0"]
	require.Equal(t, []string{"Apache-2.0", "MIT"}, merged.LicenseIDs)
	require.Equal(t, "(c) Foo Inc", merged.Copyright)
	require.Empty(t, missing)
}

func TestBuildModel_FailsOnMissingLicenses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.OutLicensesDir = t.TempDir()
	sbom := SBOM{
		Components: []Component{
			{
				Name:    "go-metrics",
				Version: "v0.0.0-20190712003943-3a3abf6ff459",
				PURL:    "pkg:golang/github.com/rcrowley/go-metrics@v0.0.0-20190712003943-3a3abf6ff459",
			},
			{
				Name:    "foo",
				Version: "1.2.3",
			},
		},
	}

	_, err := buildModel(ctx, cfg, sbom, Filters{}, nil, nil, map[string]string{})
	require.Error(t, err)

	var missingErr MissingLicensesError
	require.ErrorAs(t, err, &missingErr)
	require.ElementsMatch(t, []string{
		"pkg:golang/github.com/rcrowley/go-metrics@v0.0.0-20190712003943-3a3abf6ff459",
		"foo@1.2.3",
	}, missingErr.Components)
	require.Equal(t, "Missing license information found. Add an updated SBOM that includes these license blocks, or map them in license-corrections.json.", missingErr.Error())
}

func TestBuildIndex_DuplicateComponentListedOncePerLicense(t *testing.T) {
	t.Parallel()

	mit := []LicenseChoice{
		{License: &struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{ID: "MIT"}},
	}

	components := []Component{
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Licenses: mit},
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Copyright: "(c) Foo Inc", Licenses: mit},
	}

	byLicense, _, missing := buildIndex(components, Filters{}, nil, nil, copyrightEnricher{})

	require.Len(t, byLicense["MIT"], 1)
	// The indexed entry is the merged one, not the pre-merge copy.
	assert.Equal(t, "(c) Foo Inc", byLicense["MIT"][0].Copyright)
	require.Empty(t, missing)
}

func TestBuildIndex_IndexesEveryLicenseOfAMergedComponent(t *testing.T) {
	t.Parallel()

	components := []Component{
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Licenses: []LicenseChoice{
			{License: &struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "MIT"}},
		}},
		{Name: "foo", Version: "1.0.0", PURL: "pkg:npm/foo@1.0.0", Licenses: []LicenseChoice{
			{License: &struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "Apache-2.0"}},
		}},
	}

	byLicense, _, missing := buildIndex(components, Filters{}, nil, nil, copyrightEnricher{})

	require.Len(t, byLicense["MIT"], 1)
	require.Len(t, byLicense["Apache-2.0"], 1)
	assert.Equal(t, []string{"Apache-2.0", "MIT"}, byLicense["MIT"][0].LicenseIDs)
	require.Empty(t, missing)
}

func TestShouldIgnoreComponent(t *testing.T) {
	t.Parallel()

	filters := Filters{
		PURLRegex: []*regexp.Regexp{
			regexp.MustCompile(`use\.local`),
		},
		Suppliers: []*regexp.Regexp{
			regexp.MustCompile("^Foo$"),
		},
	}

	c1 := Component{
		PURL: "pkg:golang/github.com/some/repo",
		Supplier: &struct {
			Name string `json:"name"`
		}{Name: "Some Supplier"},
	}
	c2 := Component{
		PURL: "pkg:npm/foo@1.2.30",
		Supplier: &struct {
			Name string `json:"name"`
		}{Name: "Foo"},
	}
	c3 := Component{
		PURL: "pkg:golang/use.local/bar@v1.0.0",
		Supplier: &struct {
			Name string `json:"name"`
		}{Name: ""},
	}

	// c4 has nil supplier (e.g., cyclonedx-gomod output).
	c4 := Component{
		PURL:     "pkg:golang/github.com/other/lib@v2.0.0",
		Supplier: nil,
	}

	require.False(t, shouldIgnoreComponent(c1, filters))
	require.True(t, shouldIgnoreComponent(c2, filters))
	require.True(t, shouldIgnoreComponent(c3, filters))
	require.False(t, shouldIgnoreComponent(c4, filters))
}
