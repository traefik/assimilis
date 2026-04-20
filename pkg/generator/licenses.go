package generator

import (
	"strings"

	"github.com/aquasecurity/trivy/pkg/licensing/expression"
)

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

	// Parse the full expression (handles parentheses and AND/OR/WITH) and let
	// NormalizeForSPDX replace spaces / invalid runes inside each SimpleExpr so
	// that names like "Apache 2.0" become "Apache-2.0" before the SPDX lookup.
	parsed, err := expression.Normalize(expr, expression.NormalizeForSPDX)
	if err != nil {
		return []string{resolveSingleLicense(expr, licenseMap)}
	}

	var ids []string
	for _, lic := range collectSimpleLicenses(parsed) {
		ids = append(ids, resolveSingleLicense(lic, licenseMap))
	}

	return ids
}

// collectSimpleLicenses walks the parsed license expression and returns the
// SPDX-like string of every leaf SimpleExpr, in left-to-right order.
func collectSimpleLicenses(e expression.Expression) []string {
	switch v := e.(type) {
	case expression.SimpleExpr:
		return []string{v.String()}
	case expression.CompoundExpr:
		return append(collectSimpleLicenses(v.Left()), collectSimpleLicenses(v.Right())...)
	}

	return nil
}

// resolveSingleLicense turns one already-cleaned token into its final ID:
// licenseMap override, canonical SPDX ID, or a LicenseRef- fallback (which can
// itself be remapped by licenseMap).
func resolveSingleLicense(lic string, licenseMap map[string]string) string {
	if mapped, ok := licenseMap[lic]; ok && mapped != "" {
		return mapped
	}

	if spdxID, ok := expression.SPDXLicenseID(lic); ok {
		return spdxID
	}

	// Pass-through any LicenseRef-* the SBOM already provided.
	if strings.HasPrefix(lic, "LicenseRef-") {
		if mapped, ok := licenseMap[lic]; ok && mapped != "" {
			return mapped
		}

		return lic
	}

	licRef := "LicenseRef-" + sanitizeID(lic)
	if mapped, ok := licenseMap[licRef]; ok && mapped != "" {
		return mapped
	}

	return licRef
}

// matchLicenseOverride checks if a PURL matches any entry in license-corrections.json.
// Keys are PURL prefixes: "pkg:golang/std" matches "pkg:golang/std@go1.25.3", and
// "pkg:golang/github.com/foo/bar" matches sub-packages like
// "pkg:golang/github.com/foo/bar/v2/sub@v2.1.0".
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

	// Strip version ("@...") before prefix matching.
	if idx := strings.LastIndex(clean, "@"); idx != -1 {
		clean = clean[:idx]
	}

	// Check whether any override key is a prefix of the version-stripped PURL.
	// This handles sub-packages and Go major versions embedded in the path
	// (e.g. key "pkg:golang/github.com/nrdcg/oci-go-sdk" matches
	// "pkg:golang/github.com/nrdcg/oci-go-sdk/v65/common@v65.0.0").
	for key, id := range overrides {
		if clean == key || strings.HasPrefix(clean, key+"/") {
			return id
		}
	}

	return ""
}

func firstNonEmpty(a string, b func() string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}

	return b()
}
