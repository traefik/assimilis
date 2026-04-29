package generator

const (
	defaultHTMLFileName   = "THIRD_PARTY_LICENSES.html"
	defaultNoticeFileName = "NOTICE.md"

	embeddedLicenseMapPath         = "data/license-map.json"
	embeddedLicenseCorrectionsPath = "data/license-corrections.json"
	embeddedFiltersPath            = "data/filters.json"

	spdxNameMapURLFmt     = "https://raw.githubusercontent.com/spdx/license-list-data/%s/json/licenses.json"
	spdxLicenseTextURLFmt = "https://raw.githubusercontent.com/spdx/license-list-data/%s/text/%s.txt"
)
