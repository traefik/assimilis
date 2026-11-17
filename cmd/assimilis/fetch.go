package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/traefik/assimilis/v2/pkg/aikido"
	"github.com/traefik/assimilis/v2/pkg/generator"
	"github.com/traefik/assimilis/v2/pkg/logger"
	"github.com/urfave/cli/v3"
)

const (
	fetchTypeSBOM     = "sbom"
	fetchTypeRepoCode = "repo-code"
)

type fetchConfig struct {
	baseURL      string
	clientID     string
	clientSecret string
	repoCode     string
	fetchType    string
	repoName     string
	branch       string
}

type sbomDownloader interface {
	DownloadSBOM(ctx context.Context, repoCode string) ([]byte, error)
}

func buildFetchCommand(cfg *generator.Config) *cli.Command {
	fetchCfg := fetchConfig{baseURL: "https://app.aikido.dev", fetchType: fetchTypeSBOM}

	return &cli.Command{
		Name:  "fetch",
		Usage: "Fetch an SBOM and generate attribution files, or retrieve an Aikido repository code",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "type",
				Usage:       "What to fetch: sbom or repo-code",
				Value:       fetchCfg.fetchType,
				Sources:     cli.EnvVars("AIKIDO_FETCH_TYPE"),
				Destination: &fetchCfg.fetchType,
			},
			&cli.StringFlag{
				Name:        "repo-code",
				Usage:       "Aikido repository code (overrides automatic lookup)",
				Sources:     cli.EnvVars("AIKIDO_REPO_CODE"),
				Destination: &fetchCfg.repoCode,
			},
			&cli.StringFlag{
				Name:        "aikido-repo-name",
				Usage:       "Repository name in Aikido (defaults to --repo-name)",
				Sources:     cli.EnvVars("AIKIDO_REPO_NAME"),
				Destination: &fetchCfg.repoName,
			},
			&cli.StringFlag{
				Name:        "branch",
				Usage:       "Branch scanned by Aikido",
				Sources:     cli.EnvVars("AIKIDO_BRANCH"),
				Destination: &fetchCfg.branch,
			},
			&cli.StringFlag{
				Name:        "aikido-base-url",
				Usage:       "Aikido API base URL",
				Value:       fetchCfg.baseURL,
				Sources:     cli.EnvVars("AIKIDO_BASE_URL"),
				Destination: &fetchCfg.baseURL,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			fetchCfg.clientID = os.Getenv("AIK_CLIENT")
			fetchCfg.clientSecret = os.Getenv("AIK_SECRET")

			return runFetchCommand(ctx, *cfg, fetchCfg, command.Root().Writer)
		},
	}
}

func runFetchCommand(ctx context.Context, cfg generator.Config, fetchCfg fetchConfig, output io.Writer) error {
	if err := validateFetch(cfg, fetchCfg); err != nil {
		return err
	}

	if strings.TrimSpace(fetchCfg.repoCode) != "" && fetchCfg.fetchType == fetchTypeRepoCode {
		_, err := fmt.Fprintln(output, fetchCfg.repoCode)

		return err
	}

	client := aikido.NewClient(fetchCfg.baseURL, fetchCfg.clientID, fetchCfg.clientSecret, &http.Client{
		Timeout: 2 * time.Minute,
	})

	if strings.TrimSpace(fetchCfg.repoCode) == "" {
		repoName := fetchCfg.repoName
		if repoName == "" {
			repoName = cfg.RepoName
		}

		repoCode, err := client.FetchRepoCode(ctx, repoName, fetchCfg.branch)
		if err != nil {
			return err
		}

		fetchCfg.repoCode = repoCode
	}

	if fetchCfg.fetchType == fetchTypeRepoCode {
		_, err := fmt.Fprintln(output, fetchCfg.repoCode)

		return err
	}

	logger.Setup("info")

	return fetch(ctx, cfg, fetchCfg.repoCode, client, generator.Run, time.Now)
}

func validateFetch(cfg generator.Config, fetchCfg fetchConfig) error {
	switch fetchCfg.fetchType {
	case fetchTypeSBOM:
		if err := validate(cfg); err != nil {
			return err
		}
	case fetchTypeRepoCode:
	default:
		return fmt.Errorf("unsupported fetch type %q: expected sbom or repo-code", fetchCfg.fetchType)
	}

	if strings.TrimSpace(fetchCfg.repoCode) != "" && fetchCfg.fetchType == fetchTypeRepoCode {
		return nil
	}

	if strings.TrimSpace(fetchCfg.clientID) == "" {
		return fmt.Errorf("AIK_CLIENT cannot be empty")
	}

	if strings.TrimSpace(fetchCfg.clientSecret) == "" {
		return fmt.Errorf("AIK_SECRET cannot be empty")
	}

	return nil
}

func fetch(
	ctx context.Context,
	cfg generator.Config,
	repoCode string,
	downloader sbomDownloader,
	generate func(context.Context, generator.Config) error,
	now func() time.Time,
) error {
	log.Info().Str("repo_code", repoCode).Msg("Downloading CycloneDX SBOM from Aikido")

	sbom, err := downloader.DownloadSBOM(ctx, repoCode)
	if err != nil {
		return fmt.Errorf("failed to download SBOM: %w", err)
	}

	if err := validateSBOM(sbom); err != nil {
		return fmt.Errorf("invalid SBOM returned by Aikido: %w", err)
	}

	sbomFile := filepath.Join(cfg.SBOMPath, cfg.RepoName+".cdx.json")
	if err := writeFile(sbomFile, sbom); err != nil {
		return fmt.Errorf("failed to save SBOM: %w", err)
	}

	log.Info().Str("path", sbomFile).Msg("SBOM saved")

	if err := generate(ctx, cfg); err != nil {
		return fmt.Errorf("failed to run generator: %w", err)
	}

	timestampFile := filepath.Join(cfg.OutDir, ".last_generated_at")
	timestamp := now().UTC().Format(time.RFC3339) + "\n"

	if err := writeFile(timestampFile, []byte(timestamp)); err != nil {
		return fmt.Errorf("failed to write generation timestamp: %w", err)
	}

	log.Info().Str("timestamp_file", timestampFile).Msg("Third-party files generated successfully")

	return nil
}

func validateSBOM(data []byte) error {
	var document struct {
		BOMFormat string `json:"bomFormat"`
	}

	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	if document.BOMFormat != "CycloneDX" {
		return fmt.Errorf(
			"unexpected bomFormat %q",
			document.BOMFormat,
		)
	}

	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".assimilis-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tmpPath := tmp.Name()

	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("failed to set temporary file permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace destination file: %w", err)
	}

	return nil
}
