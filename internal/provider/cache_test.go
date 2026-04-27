// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/alzlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveAndLoadCacheFile verifies that the cache file helpers can round-trip
// an empty AlzLib's built-in cache through a gzipped file on disk.
func TestSaveAndLoadCacheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alzlib-cache.json.gz")

	alz := alzlib.NewAlzLib(nil)
	ctx := context.Background()

	// Saving with no built-ins loaded should still produce a valid cache file.
	require.NoError(t, saveCacheFile(ctx, alz, path))
	assert.FileExists(t, path)

	// Loading the file we just wrote should succeed and inject a cache.
	alz2 := alzlib.NewAlzLib(nil)
	require.NoError(t, loadCacheFile(ctx, alz2, path))
}

// TestLoadCacheFileMissingIsNoOp verifies that loading a non-existent cache file
// is treated as a no-op so that the cache file can be created on first run when
// `cache_file_save_enabled` is true.
func TestLoadCacheFileMissingIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json.gz")

	alz := alzlib.NewAlzLib(nil)
	require.NoError(t, loadCacheFile(context.Background(), alz, path))
}

// TestLoadCacheFileInvalid verifies that a non-gzip file produces an error.
func TestLoadCacheFileInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.gz")
	// Write some non-gzip data.
	require.NoError(t, os.WriteFile(path, []byte("not-a-gzip-file"), 0o600))

	alz := alzlib.NewAlzLib(nil)
	err := loadCacheFile(context.Background(), alz, path)
	assert.Error(t, err)
}

// TestSaveCacheFileOverwrite verifies that saveCacheFile can be called more
// than once against the same destination path. This guards against platform
// differences (notably Windows) where os.Rename can fail if the destination
// already exists.
func TestSaveCacheFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alzlib-cache.json.gz")

	alz := alzlib.NewAlzLib(nil)
	ctx := context.Background()

	require.NoError(t, saveCacheFile(ctx, alz, path))
	assert.FileExists(t, path)

	// A second save against the same path must succeed (overwrite).
	require.NoError(t, saveCacheFile(ctx, alz, path))
	assert.FileExists(t, path)

	// And the file should still be loadable after overwrite.
	alz2 := alzlib.NewAlzLib(nil)
	require.NoError(t, loadCacheFile(ctx, alz2, path))
}
