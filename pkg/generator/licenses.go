package generator

import (
	"regexp"
	"strings"

	"github.com/aquasecurity/trivy/pkg/licensing"
	spdxexp "github.com/aquasecurity/trivy/pkg/licensing/expression"
)

var licenseSplitRegexp = regexp.MustCompile(`(,?[_ ]+(?i:(?:or|and))[_ ]+)|(,[ ]*)`)

func normalizeLicenseIDs(licenses []LicenseChoice, licenseMap map[string]string) []string {
	var ids []string

	for _, item := range licenses {
		if item.License != nil && item.License.ID != "" {
			ids = append(ids, item.License.ID)

			continue
		}

		expr := strings.TrimSpace(firstNonEmpty(item.Expression, func() string {
			if item.License != nil {
				return item.License.Name
			}

			return ""
		}))
		if expr == "" {
			continue
		}

		if mapped, ok := licenseMap[expr]; ok && mapped != "" {
			ids = append(ids, mapped)

			continue
		}

		split := licenseSplitRegexp.Split(expr, -1)
		for _, l := range split {
			lic := licensing.Normalize(l)
			if spdxexp.ValidateSPDXLicense(lic) {
				ids = append(ids, lic)

				continue
			}

			ids = append(ids, "LicenseRef-"+sanitizeID(lic))
		}
	}

	return uniqSorted(ids)
}

func firstNonEmpty(a string, b func() string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}

	return b()
}
