package generator

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed templates/*.gotpl data/license-map.json
var embedded embed.FS

// UnknownLicensesError indicates that some license expressions could not be resolved.
type UnknownLicensesError struct {
	IDs []string
}

func (e UnknownLicensesError) Error() string {
	return "Unknown license expressions found"
}

// Run executes the generator with the given configuration.
func Run(ctx context.Context, cfg Config) error {
	sbom, licenseMap, spdxNames, err := loadInputs(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to load inputs: %w", err)
	}

	// #nosec G301 -- output directory should be readable in artifacts
	if err = os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	// #nosec G301 -- output directory should be readable in artifacts
	if err = os.MkdirAll(cfg.OutLicensesDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output licenses directory: %w", err)
	}

	model, err := buildModel(ctx, cfg, sbom, licenseMap, spdxNames)
	if err != nil {
		return fmt.Errorf("failed to build model: %w", err)
	}

	htmlOut, err := renderHTML(cfg, embedded, model)
	if err != nil {
		return fmt.Errorf("failed to render HTML output: %w", err)
	}

	noticeOut, err := renderText(cfg, embedded, model)
	if err != nil {
		return fmt.Errorf("failed to render notice output: %w", err)
	}

	tpnDir := filepath.Join(cfg.OutDir, "THIRD_PARTY_LICENSES.html")
	nDir := filepath.Join(cfg.OutDir, "NOTICE.md")
	if err := writeText(tpnDir, htmlOut); err != nil {
		return fmt.Errorf("failed to write HTML output: %w", err)
	}
	if err := writeText(nDir, noticeOut); err != nil {
		return fmt.Errorf("failed to write notice output: %w", err)
	}

	fmt.Printf("Wrote:\n- %s\n- %s\n- %s/\n",
		tpnDir,
		nDir,
		cfg.OutLicensesDir,
	)

	return nil
}

func loadInputs(ctx context.Context, cfg Config) (SBOM, map[string]string, map[string]string, error) {
	sbom, err := readJSON[SBOM](filepath.Join(cfg.SBOMPath, cfg.RepoName+".cdx.json"))
	if err != nil {
		return SBOM{}, nil, nil, fmt.Errorf("failed to read SBOM: %w", err)
	}

	licenseMap, err := loadLicenseMap(cfg.LicenseMapPath)
	if err != nil {
		return SBOM{}, nil, nil, fmt.Errorf("failed to read license map: %w", err)
	}

	spdxNames, err := loadSpdxNameMap(ctx, cfg.SPDXVersion)
	if err != nil {
		return SBOM{}, nil, nil, fmt.Errorf("failed to load SPDX names: %w", err)
	}

	return sbom, licenseMap, spdxNames, nil
}

func loadLicenseMap(path string) (map[string]string, error) {
	b, err := embedded.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read license map: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal license map: %w", err)
	}
	return m, nil
}

func buildModel(ctx context.Context, cfg Config, sbom SBOM, licenseMap map[string]string, spdxNames map[string]string) (Model, error) {
	byLicense, byKey := buildIndex(cfg, sbom.Components, licenseMap, spdxNames)

	licenses, err := buildLicenseBlocks(ctx, cfg, byLicense, spdxNames)
	if err != nil {
		return Model{}, err
	}

	overview := buildOverview(licenses)

	notices := buildNotices(byKey)

	return Model{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Overview:    overview,
		Licenses:    licenses,
		Notices:     notices,
	}, nil
}

func buildOverview(licenses []LicenseBlock) []OverviewItem {
	overview := make([]OverviewItem, 0, len(licenses))
	for _, l := range licenses {
		overview = append(overview, OverviewItem{ID: l.ID, Name: l.Name, Count: len(l.UsedBy)})
	}
	sort.Slice(overview, func(i, j int) bool {
		if overview[i].Count != overview[j].Count {
			return overview[i].Count > overview[j].Count
		}
		return overview[i].ID < overview[j].ID
	})
	return overview
}

func buildNotices(byKey map[string]OutComponent) []OutComponent {
	notices := make([]OutComponent, 0, len(byKey))
	for _, c := range byKey {
		if strings.TrimSpace(c.Copyright) != "" {
			notices = append(notices, c)
		}
	}
	sort.Slice(notices, func(i, j int) bool {
		return notices[i].Name+notices[i].Version < notices[j].Name+notices[j].Version
	})
	return notices
}

func buildLicenseBlocks(ctx context.Context, cfg Config, byLicense map[string][]OutComponent, spdxNames map[string]string) ([]LicenseBlock, error) {
	licenseIDs := make([]string, 0, len(byLicense))
	for id := range byLicense {
		licenseIDs = append(licenseIDs, id)
	}
	sort.Strings(licenseIDs)

	licenses := make([]LicenseBlock, 0, len(licenseIDs))
	var unknowns []string
	for _, id := range licenseIDs {
		comps := byLicense[id]
		sort.Slice(comps, func(i, j int) bool {
			return comps[i].Name+comps[i].Version < comps[j].Name+comps[j].Version
		})

		name := spdxNames[id]
		if name == "" {
			name = id
			if tmp, ok := strings.CutPrefix(id, "LicenseRef-"); ok {
				name = strings.ReplaceAll(tmp, "-", " ") + " (custom license)"
			}
		}

		var text string
		t, errl := getLicenseText(ctx, cfg, id)
		if errl != nil {
			unknowns = append(unknowns, id)
			text = fmt.Sprintf("ERROR: Could not retrieve license text for %s: %v", id, errl)
		} else {
			text = t
		}

		licenses = append(licenses, LicenseBlock{
			ID:     id,
			Name:   name,
			Text:   text,
			UsedBy: comps,
		})
	}

	if len(unknowns) > 0 {
		return nil, UnknownLicensesError{IDs: unknowns}
	}

	return licenses, nil
}

func buildIndex(cfg Config, components []Component, licenseMap map[string]string, spdxNames map[string]string) (map[string][]OutComponent, map[string]OutComponent) {
	byLicense := map[string][]OutComponent{}
	byKey := map[string]OutComponent{}

	for _, c := range components {
		if shouldIgnorePURL(cfg, c.PURL) {
			continue
		}

		ids := normalizeLicenseIDs(c.Licenses, licenseMap, spdxNames)

		out := OutComponent{
			Name:       c.Name,
			Version:    c.Version,
			PURL:       c.PURL,
			URL:        componentURLFromPurl(c.PURL),
			LicenseIDs: ids,
			Copyright:  c.Copyright,
		}

		key := c.PURL
		if key == "" {
			key = c.Name + "@" + c.Version
		}

		if existing, ok := byKey[key]; ok {
			existing.LicenseIDs = uniqSorted(append(existing.LicenseIDs, out.LicenseIDs...))
			if existing.Copyright == "" && out.Copyright != "" {
				existing.Copyright = out.Copyright
			}
			byKey[key] = existing
			out = existing
		} else {
			byKey[key] = out
		}

		for _, id := range ids {
			byLicense[id] = append(byLicense[id], out)
		}
	}

	return byLicense, byKey
}
