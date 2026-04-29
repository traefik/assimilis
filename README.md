# Myrmica Assimilis - Generate OSS Attribution Files

[![Static Badge](https://img.shields.io/badge/%F0%9F%90%9C_What%20does%20Myrmica%20Assimilis%20mean%3F-white?style=plastic)](https://antwiki.org/wiki/Myrmica_assimilis)

## Description

Generate third-party attribution artifacts (NOTICE + "Third Party Licenses" HTML) from a **CycloneDX JSON SBOM**.

It is intended to be used in CI/CD to produce release artifacts that can be shipped alongside binaries/images.

### Output

By default, it writes:

- `third_party/THIRD_PARTY_LICENSES.html`: grouped by license, with license texts and "used by" list. Based on [cargo-about](https://github.com/EmbarkStudios/cargo-about) (_default example available [here](https://embarkstudios.github.io/cargo-about/cli/generate/default-example.html)_)
- `third_party/NOTICE.md`: per-dependency copyright/notice block (_only for deps that expose copyright_)
- `third_party/licenses/*.txt`: cached SPDX license texts

## Usage

1. Place the SBOM in `third_party/sbom`

    By default, Assimilis looks for `third_party/sbom/<REPO_NAME>.cdx.json`. The SBOM must have this exact naming pattern.

2. Run Assimilis

    From your repository root:

    ```bash
    assimilis --repo-name <REPO_NAME>
    ```

### Configuration

```
NAME:
   assimilis - Generate OSS attribution files

USAGE:
   assimilis [global options] [command [command options]]

COMMANDS:
   version  Display version information
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --repo-name string          Name of the repository
   --output-dir string         Base output directory (default: "third_party")
   --html-template string      Override HTML template path (default: embedded)
   --notice-template string    Override NOTICE template path (default: embedded)
   --spdx-version string       SPDX license-list-data version/tag (default: "v3.27.0")
   --html-filename string      Output HTML filename (default: "THIRD_PARTY_LICENSES.html")
   --notice-filename string    Output NOTICE filename (default: "NOTICE.md")
   --license-map string        Path to external license-map JSON (default: embedded)
   --license-corrections string   Path to external license-corrections JSON (default: embedded)
   --filters string            Path to external filters JSON (default: embedded)
   --help, -h                  show help
```

### License Map

Assimilis ships with an embedded `license-map.json` that normalizes non-standard license expressions to SPDX IDs (e.g. `"Python Software Foundation License"` → `"PSF-2.0"`). To provide your own, use `--license-map path/to/license-map.json`.

### Missing Licenses

Assimilis can apply per-PURL license corrections via `license-corrections.json`. Entries take priority over whatever the SBOM reported, so they can both fill in absent licenses (when the SBOM generator failed to detect one) and correct wrong ones (when the SBOM generator reported an incorrect license). The embedded `license-corrections.json` covers known gaps. To provide your own, use `--license-corrections path/to/license-corrections.json`.

Example:

```json
{
    "pkg:golang/std": "BSD-3-Clause",
    "pkg:npm/config-chain": "MIT"
}
```

Keys are matched as PURL prefixes — `"pkg:golang/std"` matches `"pkg:golang/std@go1.25.3"`, and `"pkg:golang/github.com/foo/bar"` matches sub-packages like `"pkg:golang/github.com/foo/bar/v2/sub@v2.1.0"`.

### Custom/Non-SPDX Licenses (LicenseRef-*)

If a component uses a non-SPDX license ID or an unmapped license expression, Assimilis expects a corresponding license text file in `third_party/licenses/custom`.

Example:

```bash
third_party/licenses/custom/LicenseRef-<CUSTOM_LICENSE_NAME>.txt
```

If the text is missing, generation fails.

## The Mymirca colony

- [Myrmica Lobicornis](https://github.com/traefik/lobicornis) 🐜: Update and merge pull requests.
- [Myrmica Aloba](https://github.com/traefik/aloba) 🐜: Add labels and milestone on pull requests and issues.
- [Messor Structor](https://github.com/traefik/structor) 🐜: Manage multiple documentation versions with Mkdocs.
- [Lasius Mixtus](https://github.com/traefik/mixtus) 🐜: Publish documentation to a GitHub repository from another.
- [Myrmica Bibikoffi](https://github.com/traefik/bibikoffi) 🐜: Closes stale issues
- [Chalepoxenus Kutteri](https://github.com/traefik/kutteri) 🐜: Track a GitHub repository and publish on Slack.
- [Myrmica Gallienii](https://github.com/traefik/gallienii) 🐜: Keep Forks Synchronized
