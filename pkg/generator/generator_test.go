package generator

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnknownLicensesError_IncludesIDs(t *testing.T) {
	t.Parallel()

	err := UnknownLicensesError{IDs: []string{"LicenseRef-MIT-X11", "LicenseRef-Unknown"}}
	msg := err.Error()

	require.Contains(t, msg, "LicenseRef-MIT-X11")
	require.Contains(t, msg, "LicenseRef-Unknown")
	require.Contains(t, msg, "custom")
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

	byLicense, byKey := buildIndex(components, Filters{}, nil, overrides, copyrightEnricher{})

	require.Contains(t, byLicense, "BSD-3-Clause")
	require.Contains(t, byLicense, "MIT")
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

	_, byKey := buildIndex(components, Filters{}, nil, overrides, copyrightEnricher{})

	// missing-licenses entries take priority and correct wrong licenses from the SBOM.
	require.Equal(t, []string{"MIT"}, byKey["pkg:npm/foo@1.0.0"].LicenseIDs)
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

	_, byKey := buildIndex(components, Filters{}, nil, nil, copyrightEnricher{})

	merged := byKey["pkg:npm/foo@1.0.0"]
	require.Equal(t, []string{"Apache-2.0", "MIT"}, merged.LicenseIDs)
	require.Equal(t, "(c) Foo Inc", merged.Copyright)
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
