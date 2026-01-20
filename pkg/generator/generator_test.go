package generator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
