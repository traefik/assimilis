package generator

import (
	"strings"

	spdxexp "github.com/khulnasoft/tunnel/pkg/licensing"
)

func normalizeLicenseIDs(licenses []LicenseChoice, licenseMap map[string]string, spdxNames map[string]string) []string {
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

		split := spdxexp.SplitLicenses(strings.ToLower(expr))
		for _, l := range split {
			lic := spdxexp.Normalize(l)
			if extracted := spdxexp.Normalize(lic); spdxNames[extracted] != "" {
				ids = append(ids, extracted)
				continue
			}
			ids = append(ids, "LicenseRef-"+sanitizeID(expr, 40))
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
