package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/traefik/assimilis/pkg/generator"
)

func TestValidate_Fail(t *testing.T) {
	t.Parallel()

	cfg := generator.DefaultConfig()
	cfg.RepoName = ""

	err := validate(cfg)
	require.Error(t, err)
}

func TestValidate_OK(t *testing.T) {
	t.Parallel()

	cfg := generator.DefaultConfig()
	cfg.RepoName = "repo"

	err := validate(cfg)
	require.NoError(t, err)
}
