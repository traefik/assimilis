package generator

import "regexp"

// SBOM represents a CycloneDX SBOM structure.
type SBOM struct {
	Components []Component `json:"components"`
}

// Component represents a component in the SBOM.
type Component struct {
	Name      string          `json:"name"`
	Version   string          `json:"version"`
	PURL      string          `json:"purl"`
	Copyright string          `json:"copyright"`
	Supplier  string          `json:"supplier"`
	Licenses  []LicenseChoice `json:"licenses"`
}

// Filters holds compiled regex patterns for excluding components.
type Filters struct {
	PURLRegex []*regexp.Regexp `json:"purlRegex"`
	Suppliers []*regexp.Regexp `json:"suppliers"`
}

// LicenseChoice represents a license in a component.
type LicenseChoice struct {
	Expression string `json:"expression"`
	License    *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"license"`
}

// OutComponent represents a component in the output model.
type OutComponent struct {
	Name       string
	Version    string
	PURL       string
	URL        string
	LicenseIDs []string
	Copyright  string
}

// LicenseBlock represents a license block in the output model.
type LicenseBlock struct {
	ID     string
	Name   string
	Text   string
	UsedBy []OutComponent
}

// OverviewItem represents an overview item in the output model.
type OverviewItem struct {
	ID    string
	Name  string
	Count int
}

// Model represents the data model for the output.
type Model struct {
	GeneratedAt string
	Overview    []OverviewItem
	Licenses    []LicenseBlock
	Notices     []OutComponent
}
