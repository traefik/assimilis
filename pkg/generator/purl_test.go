package generator

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldIgnorePURL(t *testing.T) {
	t.Parallel()

	filters := Filters{}
	filters.PURLRegex = []*regexp.Regexp{regexp.MustCompile(`use\.local`)}

	assert.False(t, shouldIgnorePURL(filters, ""))
	assert.False(t, shouldIgnorePURL(filters, "pkg:npm/foo@1.2.30"))
	assert.True(t, shouldIgnorePURL(filters, "pkg:golang/use.local/bar@v1.0.0"))
}

func TestComponentURLFromPurl(t *testing.T) {
	t.Parallel()

	assert.Empty(t, componentURLFromPurl(""))
	assert.Empty(t, componentURLFromPurl("not-a-purl"))
	assert.Equal(t, "https://www.npmjs.com/package/foo", componentURLFromPurl("pkg:npm/foo@1.2.30"))
	assert.Equal(t, "https://pypi.org/project/bar/", componentURLFromPurl("pkg:pypi/bar@2.3.4"))

	assert.Equal(t, "https://github.com/traefik/traefik", componentURLFromPurl("pkg:golang/github.com/traefik/traefik@v3.6.0"))
	assert.Equal(t, "https://pkg.go.dev/golang.org/x/baz", componentURLFromPurl("pkg:golang/golang.org/x/baz@v4.5.6"))
	assert.Empty(t, componentURLFromPurl("pkg:maven/com.example/qux@1.0.0"))
}
