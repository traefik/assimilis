package generator

import (
	"regexp"
	"strings"
)

var purlRegex = regexp.MustCompile(`^pkg:([^/]+)/(.+)@([^@]+)$`)

func shouldIgnorePURL(filters Filters, purl string) bool {
	if purl == "" {
		return false
	}

	for _, re := range filters.PURLRegex {
		if re.MatchString(purl) {
			return true
		}
	}

	return false
}

func componentURLFromPurl(purl string) string {
	m := purlRegex.FindStringSubmatch(purl)
	if len(m) != 4 {
		return ""
	}

	typ := m[1]
	name := m[2]

	switch typ {
	case "npm":
		return "https://www.npmjs.com/package/" + name
	case "pypi":
		return "https://pypi.org/project/" + name + "/"
	case "golang":
		if strings.HasPrefix(name, "github.com/") {
			parts := strings.Split(name, "/")
			if len(parts) >= 3 {
				return "https://github.com/" + parts[1] + "/" + parts[2]
			}
		}

		return "https://pkg.go.dev/" + name
	default:
		return ""
	}
}
