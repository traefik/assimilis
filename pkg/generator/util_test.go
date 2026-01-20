package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniqSorted(t *testing.T) {
	t.Parallel()

	in := []string{" b ", "a", "", "a", "c", "B", "  "}
	out := uniqSorted(in)
	assert.Equal(t, []string{"B", "a", "b", "c"}, out)
}

func TestSanitizeID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "MIT", sanitizeID("MIT"))
	assert.Equal(t, "Apache-2-0", sanitizeID("Apache 2.0"))
	assert.Equal(t, "MIT-AND-Apache-2-0", sanitizeID("MIT AND Apache 2.0"))
}

func TestWriteText_CreateDirAndWriteFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "foo", "bar", "baz.txt")

	require.NoError(t, writeText(outPath, "test"))

	b, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Equal(t, "test", string(b))
}

func TestReadJSON(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	p := filepath.Join(tmp, "bom.json")
	require.NoError(t, os.WriteFile(p, []byte(`{"components":[{"name":"foo","version":"1"}]}`), 0o644))

	sbom, err := readJSON[SBOM](p)
	require.NoError(t, err)
	require.Len(t, sbom.Components, 1)
	assert.Equal(t, "foo", sbom.Components[0].Name)
	assert.Equal(t, "1", sbom.Components[0].Version)
}
