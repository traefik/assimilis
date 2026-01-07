// Package generator generates NOTICE/HTML attribution artifacts from a CycloneDX SBOM.
package generator

import "regexp"

// Config the global configuration.
type Config struct {
	RepoName string

	SBOMPath         string
	LicenseMapPath   string
	HTMLTemplatePath string
	NoticeTplPath    string

	OutDir           string
	OutLicensesDir   string
	CustomLicenseDir string

	SPDXVersion string

	IgnorePURLPatterns []*regexp.Regexp
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	outDir := "third_party"
	outLicensesDir := outDir + "/licenses"
	customLicenseDir := outLicensesDir + "/custom"

	return Config{
		SBOMPath:       "compliance/sbom",
		LicenseMapPath: "compliance/config/license-map.json",

		OutDir:           outDir,
		OutLicensesDir:   outLicensesDir,
		CustomLicenseDir: customLicenseDir,

		SPDXVersion: "v3.27.0",

		IgnorePURLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`use\.local`),
		},
	}
}
