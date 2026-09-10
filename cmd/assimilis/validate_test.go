package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseMaxAge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value    string
		expected time.Duration
		wantErr  bool
	}{
		{value: "90m", expected: 90 * time.Minute},
		{value: "12h", expected: 12 * time.Hour},
		{value: "2d", expected: 48 * time.Hour},
		{value: " 2d ", expected: 48 * time.Hour},
		{value: "0h", wantErr: true},
		{value: "1.5h", wantErr: true},
		{value: "1w", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.value, func(t *testing.T) {
			t.Parallel()

			result, err := parseMaxAge(testCase.value)

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expected, result)
		})
	}
}

func TestValidateGenerationAge(t *testing.T) {
	t.Parallel()

	commitAt := time.Date(2026, time.July, 15, 12, 15, 0, 0, time.UTC)

	testCases := []struct {
		name        string
		generatedAt time.Time
		maxAge      time.Duration
		expectedAge time.Duration
		wantErr     bool
	}{
		{
			name:        "recent",
			generatedAt: commitAt.Add(-15 * time.Minute),
			maxAge:      time.Hour,
			expectedAge: 15 * time.Minute,
		},
		{
			name:        "exact maximum age",
			generatedAt: commitAt.Add(-12 * time.Hour),
			maxAge:      12 * time.Hour,
			expectedAge: 12 * time.Hour,
		},
		{
			name:        "too old",
			generatedAt: commitAt.Add(-13 * time.Hour),
			maxAge:      12 * time.Hour,
			wantErr:     true,
		},
		{
			name:        "future timestamp",
			generatedAt: commitAt.Add(time.Minute),
			maxAge:      12 * time.Hour,
			wantErr:     true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			age, err := validateGenerationAge(
				testCase.generatedAt,
				commitAt,
				testCase.maxAge,
			)

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expectedAge, age)
		})
	}
}

func TestValidateThirdParty(t *testing.T) {
	t.Parallel()

	commitAt := time.Date(2026, time.July, 15, 12, 15, 0, 0, time.UTC)

	testCases := []struct {
		name        string
		generatedAt string
		expectedAge time.Duration
		wantErr     bool
	}{
		{
			name:        "valid RFC3339 timestamp",
			generatedAt: "2026-07-15T12:11:31Z",
			expectedAge: 3*time.Minute + 29*time.Second,
		},
		{
			name:        "legacy Unix timestamp",
			generatedAt: "1786714804",
			wantErr:     true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			timestampFile := filepath.Join(repoRoot, "third_party", ".last_generated_at")

			age, err := validateThirdParty(
				context.Background(),
				timestampFile,
				12*time.Hour,
				"HEAD",
				fakeValidationGit(t, repoRoot, commitAt, testCase.generatedAt),
			)

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expectedAge, age)
		})
	}
}

// An ambiguous "v2" makes rev-parse warn on stderr while exiting 0.
func TestRunGit_IgnoresStderrOnSuccess(t *testing.T) {
	t.Parallel()

	repoRoot := newAmbiguousRefRepo(t)

	ctx := context.Background()

	// Without that warning the test would still pass on a runGit that merges
	// stderr into stdout, so the precondition is asserted rather than assumed.
	require.NotEmpty(
		t,
		gitStderr(t, repoRoot, "rev-parse", "--verify", "v2^{commit}"),
		"git no longer warns about the ambiguous ref this test relies on",
	)

	commit, err := runGit(ctx, repoRoot, "rev-parse", "--verify", "v2^{commit}")
	require.NoError(t, err)
	require.Regexp(t, "^[0-9a-f]{40}$", commit)

	// The sha must stay usable by the commands the caller chains onto it.
	_, err = runGit(ctx, repoRoot, "show", "-s", "--format=%ct", commit)
	require.NoError(t, err)
}

func TestRunGit_ReportsStderrOnFailure(t *testing.T) {
	t.Parallel()

	repoRoot := newAmbiguousRefRepo(t)

	_, err := runGit(
		context.Background(),
		repoRoot,
		"rev-parse",
		"--verify",
		"does-not-exist^{commit}",
	)
	// Not asserted on git's wording: its messages are localized.
	require.ErrorContains(t, err, "exit status 128: ")
}

// newAmbiguousRefRepo: one commit, reachable as both branch "v2" and tag "v2".
func newAmbiguousRefRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()

	commands := [][]string{
		{"init", "-b", "main"},
		{"config", "user.name", "assimilis-test"},
		{"config", "user.email", "assimilis-test@example.invalid"},
		{"commit", "--allow-empty", "-m", "initial"},
		{"branch", "v2"},
		{"tag", "v2"},
	}

	for _, args := range commands {
		cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)

		output, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, output)
	}

	return repoRoot
}

// gitStderr: git's stderr for one command, kept apart from stdout.
func gitStderr(t *testing.T, repoRoot string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	require.NoErrorf(t, cmd.Run(), "git %v: %s", args, stderr.String())

	return strings.TrimSpace(stderr.String())
}

func fakeValidationGit(t *testing.T, repoRoot string, commitAt time.Time, generatedAt string) gitRunner {
	t.Helper()

	return func(
		_ context.Context,
		dir string,
		args ...string,
	) (string, error) {
		t.Helper()

		switch strings.Join(args, " ") {
		case "rev-parse --show-toplevel":
			require.Empty(t, dir)

			return repoRoot, nil

		case "rev-parse --verify HEAD^{commit}":
			require.Equal(t, repoRoot, dir)

			return "testReleaseCommit", nil

		case "show -s --format=%ct " + "testReleaseCommit":
			require.Equal(t, repoRoot, dir)

			return strconv.FormatInt(commitAt.Unix(), 10), nil

		case "cat-file blob " +
			"testReleaseCommit" +
			":third_party/.last_generated_at":
			require.Equal(t, repoRoot, dir)

			return generatedAt + "\n", nil

		default:
			return "", fmt.Errorf("unexpected git command: %v", args)
		}
	}
}
