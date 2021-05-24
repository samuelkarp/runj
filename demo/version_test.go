package demo

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreeBSDVersion(t *testing.T) {
	version, err := FreeBSDVersion(context.Background())
	require.NoError(t, err)
	t.Log(version)
	assert.Equal(t, strings.TrimSpace(version), version)
}

func TestFreeBSDArch(t *testing.T) {
	arch, err := FreeBSDArch(context.Background())
	require.NoError(t, err)
	t.Log(arch)
	assert.Equal(t, strings.TrimSpace(arch), arch)
}
