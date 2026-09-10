package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/traefik/assimilis/v2/pkg/generator"
	"github.com/traefik/assimilis/v2/pkg/logger"
	"github.com/urfave/cli/v3"
)

const generatedAtLayout = "2006-01-02T15:04:05Z"

type validateCommandConfig struct {
	maxAge        string
	releaseRef    string
	timestampFile string
}

type gitRunner func(ctx context.Context, dir string, args ...string) (string, error)

func buildValidateCommand(cfg *generator.Config) *cli.Command {
	validateCfg := validateCommandConfig{
		maxAge:     "12h",
		releaseRef: "HEAD",
	}

	return &cli.Command{
		Name:  "validate",
		Usage: "Validate that third-party files were generated recently before a release commit",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "max-age",
				Usage:       "Maximum age as a positive integer followed by m, h, or d (for example 90m, 12h, or 2d)",
				Value:       validateCfg.maxAge,
				Sources:     cli.EnvVars("MAX_AGE"),
				Destination: &validateCfg.maxAge,
			},
			&cli.StringFlag{
				Name:        "release-ref",
				Usage:       "Git commit, tag, or branch to validate",
				Value:       validateCfg.releaseRef,
				Sources:     cli.EnvVars("RELEASE_REF"),
				Destination: &validateCfg.releaseRef,
			},
			&cli.StringFlag{
				Name:        "timestamp-file",
				Usage:       "Timestamp file relative to the repository root (default: third_party/.last_generated_at)",
				Sources:     cli.EnvVars("TIMESTAMP_FILE"),
				Destination: &validateCfg.timestampFile,
			},
		},
		Action: func(ctx context.Context, _ *cli.Command) error {
			return runValidateCommand(ctx, cfg, validateCfg)
		},
	}
}

func runValidateCommand(ctx context.Context, cfg *generator.Config, validateCfg validateCommandConfig) error {
	logger.Setup("info")

	maxAge, err := parseMaxAge(validateCfg.maxAge)
	if err != nil {
		return err
	}

	timestampFile := strings.TrimSpace(validateCfg.timestampFile)
	if timestampFile == "" {
		timestampFile = filepath.Join(cfg.OutDir, ".last_generated_at")
	}

	age, err := validateThirdParty(
		ctx,
		timestampFile,
		maxAge,
		validateCfg.releaseRef,
		runGit,
	)
	if err != nil {
		return err
	}

	log.Info().
		Str("release_ref", validateCfg.releaseRef).
		Str("age_at_commit", age.String()).
		Str("max_age", maxAge.String()).
		Msg("Third-party generation timestamp is valid")

	return nil
}

func validateThirdParty(ctx context.Context, timestampFile string, maxAge time.Duration, releaseRef string, git gitRunner) (time.Duration, error) {
	releaseRef = strings.TrimSpace(releaseRef)
	if releaseRef == "" {
		return 0, fmt.Errorf("--release-ref cannot be empty")
	}

	repoRoot, err := git(ctx, "", "rev-parse", "--show-toplevel")
	if err != nil {
		return 0, fmt.Errorf("failed to locate Git repository: %w", err)
	}

	timestampPath, err := repositoryRelativePath(repoRoot, timestampFile)
	if err != nil {
		return 0, err
	}

	releaseCommit, err := git(
		ctx,
		repoRoot,
		"rev-parse",
		"--verify",
		releaseRef+"^{commit}",
	)
	if err != nil {
		return 0, fmt.Errorf(
			"unable to resolve release ref %q to a commit: %w",
			releaseRef,
			err,
		)
	}

	log.Info().
		Str("release_ref", releaseRef).
		Str("release_commit", releaseCommit).
		Str("timestamp_file", timestampPath).
		Msg("Checking third-party generation timestamp")

	commitAt, err := readCommitTime(ctx, git, repoRoot, releaseCommit)
	if err != nil {
		return 0, err
	}

	generatedAt, err := readGeneratedAt(
		ctx,
		git,
		repoRoot,
		releaseCommit,
		releaseRef,
		timestampPath,
	)
	if err != nil {
		return 0, err
	}

	return validateGenerationAge(generatedAt, commitAt, maxAge)
}

func readCommitTime(ctx context.Context, git gitRunner, repoRoot string, releaseCommit string) (time.Time, error) {
	value, err := git(
		ctx,
		repoRoot,
		"show",
		"-s",
		"--format=%ct",
		releaseCommit,
	)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"failed to read timestamp for release commit %s: %w",
			releaseCommit,
			err,
		)
	}

	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid timestamp %q for release commit %s: %w",
			value,
			releaseCommit,
			err,
		)
	}

	return time.Unix(seconds, 0).UTC(), nil
}

func readGeneratedAt(ctx context.Context, git gitRunner, repoRoot string, releaseCommit string, releaseRef string, timestampPath string) (time.Time, error) {
	value, err := git(
		ctx,
		repoRoot,
		"cat-file",
		"blob",
		releaseCommit+":"+timestampPath,
	)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"timestamp file %q is not committed in %q (%s); run 'make third-party' and commit the generated files: %w",
			timestampPath,
			releaseRef,
			releaseCommit,
			err,
		)
	}

	value = strings.TrimSpace(value)

	generatedAt, err := time.Parse(generatedAtLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"invalid timestamp %q in %s at %s: expected format %s: %w",
			value,
			timestampPath,
			releaseRef,
			generatedAtLayout,
			err,
		)
	}

	return generatedAt, nil
}

func validateGenerationAge(generatedAt time.Time, commitAt time.Time, maxAge time.Duration) (time.Duration, error) {
	age := commitAt.Sub(generatedAt)

	if age < 0 {
		return 0, fmt.Errorf(
			"third-party generation timestamp %s is later than release commit timestamp %s",
			generatedAt.Format(time.RFC3339),
			commitAt.Format(time.RFC3339),
		)
	}

	if age > maxAge {
		return 0, fmt.Errorf(
			"third-party files were generated %s before the release commit; maximum age is %s. "+
				"Run 'make third-party', commit the generated files, and recreate the tag",
			age,
			maxAge,
		)
	}

	return age, nil
}

func parseMaxAge(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)

	if len(value) < 2 {
		return 0, fmt.Errorf(
			"--max-age must be a positive integer followed by m, h, or d",
		)
	}

	amount, err := strconv.ParseInt(value[:len(value)-1], 10, 64)
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf(
			"--max-age must be a positive integer followed by m, h, or d",
		)
	}

	var unit time.Duration

	switch value[len(value)-1] {
	case 'm':
		unit = time.Minute
	case 'h':
		unit = time.Hour
	case 'd':
		unit = 24 * time.Hour
	default:
		return 0, fmt.Errorf(
			"unsupported --max-age unit %q; use m, h, or d",
			string(value[len(value)-1]),
		)
	}

	const maxDuration = time.Duration(1<<63 - 1)
	if amount > int64(maxDuration/unit) {
		return 0, fmt.Errorf("--max-age %q is too large", value)
	}

	return time.Duration(amount) * unit, nil
}

func repositoryRelativePath(repoRoot, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("timestamp file cannot be empty")
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}

	relativePath, err := filepath.Rel(repoRoot, filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf(
			"failed to resolve timestamp file relative to repository: %w",
			err,
		)
	}

	if relativePath == ".." ||
		strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) ||
		filepath.IsAbs(relativePath) {
		return "", fmt.Errorf(
			"timestamp file must be inside the Git repository: %s",
			path,
		)
	}

	return filepath.ToSlash(relativePath), nil
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}

	cmd := exec.CommandContext(ctx, "git", args...)

	// Kept apart from stdout: git warns on stderr while exiting 0.
	// Callers parse stdout as a sha, a path or a blob.
	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	output, err := cmd.Output()

	details := strings.TrimSpace(stderr.String())

	if err != nil {
		if details != "" {
			return "", fmt.Errorf("%w: %s", err, details)
		}

		return "", err
	}

	if details != "" {
		log.Warn().
			Strs("args", args).
			Str("stderr", details).
			Msg("Git reported a warning")
	}

	return strings.TrimSpace(string(output)), nil
}
