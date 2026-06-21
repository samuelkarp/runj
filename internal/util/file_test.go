package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	contents := []byte("hello runj")
	require.NoError(t, os.WriteFile(source, contents, 0644))

	dest := filepath.Join(dir, "dest")
	err := CopyFile(source, dest, 0600)
	require.NoError(t, err)

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, contents, got)

	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestCopyFileMissingSource(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dest"), 0600)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestCopyFileExistingDest(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	require.NoError(t, os.WriteFile(source, []byte("new"), 0644))
	dest := filepath.Join(dir, "dest")
	require.NoError(t, os.WriteFile(dest, []byte("original"), 0644))

	// CopyFile opens the destination with O_EXCL and must refuse to clobber an
	// existing file.
	err := CopyFile(source, dest, 0600)
	assert.ErrorIs(t, err, os.ErrExist)

	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("original"), got, "destination must be left untouched")
}
