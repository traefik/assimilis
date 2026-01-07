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

```yaml
NAME:
   assimilis - Generate OSS attribution files

USAGE:
   assimilis [global options] [command [command options]]

COMMANDS:
   version  Display version information
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --repo-name string        Name of the repository
   --html-template string    Override HTML template path (default: embedded)
   --notice-template string  Override NOTICE template path (default: embedded)
   --spdx-version string     SPDX license-list-data version/tag (default: "v3.27.0")
   --help, -h                show help
```

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
