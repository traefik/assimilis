package generator

import (
	"regexp"
	"strings"

	"github.com/aquasecurity/trivy/pkg/licensing/expression"
)

var licenseSplitRegexp = regexp.MustCompile(`(,?[_ ]+(?i:(?:or|and))[_ ]+)|(,[ ]*)`)

func normalizeLicenseIDs(licenses []LicenseChoice, licenseMap map[string]string) []string {
	var ids []string

	for _, item := range licenses {
		if item.License != nil && item.License.ID != "" {
			id := item.License.ID

			// If we have a mapping for this ID (e.g. LicenseRef-MIT-X11 -> MIT), use the mapped value.
			if mapped, ok := licenseMap[id]; ok && mapped != "" {
				id = mapped
			}

			ids = append(ids, id)

			continue
		}

		ids = append(ids, resolveExpression(item, licenseMap)...)
	}

	return uniqSorted(ids)
}

func resolveExpression(item LicenseChoice, licenseMap map[string]string) []string {
	expr := strings.TrimSpace(firstNonEmpty(item.Expression, func() string {
		if item.License != nil {
			return item.License.Name
		}

		return ""
	}))
	if expr == "" {
		return nil
	}

	if mapped, ok := licenseMap[expr]; ok && mapped != "" {
		return []string{mapped}
	}

	var ids []string

	for _, l := range licenseSplitRegexp.Split(expr, -1) {
		normalized := normalizeSPDX(l)
		if spdxID, ok := expression.SPDXLicenseID(normalized); ok {
			ids = append(ids, spdxID)

			continue
		}

		licRef := "LicenseRef-" + sanitizeID(l)
		if mapped, ok := licenseMap[licRef]; ok && mapped != "" {
			ids = append(ids, mapped)
		} else {
			ids = append(ids, licRef)
		}
	}

	return ids
}

// matchLicenseOverride checks if a PURL matches any entry in license overrides.
// Override keys are PURL prefixes: "pkg:golang/std" matches "pkg:golang/std@go1.25.3".
func matchLicenseOverride(purl string, overrides map[string]string) string {
	if overrides == nil {
		return ""
	}

	// Strip qualifiers (everything after "?") for cleaner matching.
	clean := purl
	if idx := strings.Index(clean, "?"); idx != -1 {
		clean = clean[:idx]
	}

	// Try exact match first.
	if id, ok := overrides[clean]; ok {
		return id
	}

	// Try prefix match: strip version ("@...") and check.
	if idx := strings.LastIndex(clean, "@"); idx != -1 {
		prefix := clean[:idx]
		if id, ok := overrides[prefix]; ok {
			return id
		}
	}

	return ""
}

func normalizeSPDX(license string) string {
	expr, err := expression.Normalize(license)
	if err != nil {
		return license
	}

	return expression.NormalizeForSPDX(expr).String()
}

func firstNonEmpty(a string, b func() string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}

	return b()
}
