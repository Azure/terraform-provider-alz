// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package clients

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Azure/alzlib"
	"github.com/Azure/alzlib/assets"
	"github.com/Azure/alzlib/cache"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBuiltInName = "00000000-0000-0000-0000-000000000001"

// TestSaveAndLoadCacheFile verifies that the cache file helpers can round-trip
// an empty AlzLib's built-in cache through a gzipped file on disk.
func TestSaveAndLoadCacheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alzlib-cache.json.gz")

	alz := alzlib.NewAlzLib(nil)
	ctx := context.Background()

	// Saving with no built-ins loaded should still produce a valid cache file.
	require.NoError(t, SaveCacheFile(ctx, alz, path))
	assert.FileExists(t, path)

	// Loading the file we just wrote should succeed and inject a cache.
	alz2 := alzlib.NewAlzLib(nil)
	require.NoError(t, LoadCacheFile(ctx, alz2, path))
}

// TestLoadCacheFileMissingIsNoOp verifies that loading a non-existent cache file
// is treated as a no-op so that the cache file can be created on first run when
// `cache_file_save_enabled` is true.
func TestLoadCacheFileMissingIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json.gz")

	alz := alzlib.NewAlzLib(nil)
	require.NoError(t, LoadCacheFile(context.Background(), alz, path))
}

// TestLoadCacheFileInvalid verifies that a non-gzip file produces an error.
func TestLoadCacheFileInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.gz")
	// Write some non-gzip data.
	require.NoError(t, os.WriteFile(path, []byte("not-a-gzip-file"), 0o600))

	alz := alzlib.NewAlzLib(nil)
	err := LoadCacheFile(context.Background(), alz, path)
	assert.Error(t, err)
}

// TestSaveCacheFileOverwrite verifies that SaveCacheFile can be called more
// than once against the same destination path. This guards against platform
// differences (notably Windows) where os.Rename can fail if the destination
// already exists.
func TestSaveCacheFileOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alzlib-cache.json.gz")

	alz := alzlib.NewAlzLib(nil)
	ctx := context.Background()

	require.NoError(t, SaveCacheFile(ctx, alz, path))
	assert.FileExists(t, path)

	// A second save against the same path must succeed (overwrite).
	require.NoError(t, SaveCacheFile(ctx, alz, path))
	assert.FileExists(t, path)

	// And the file should still be loadable after overwrite.
	alz2 := alzlib.NewAlzLib(nil)
	require.NoError(t, LoadCacheFile(ctx, alz2, path))
}

// TestSaveCacheFileExportsOnlyLoadedDefinitions documents why the cache cannot be saved when the
// provider is configured: a loaded cache is read lazily, so right after Init the AlzLib holds no
// built-in definitions and saving would overwrite a good cache with an empty one
// (Azure/Azure-Landing-Zones#4328).
func TestSaveCacheFileExportsOnlyLoadedDefinitions(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "alzlib-cache.json.gz")
	writeSeedCache(t, path)

	alz := newTestAlzLib(t)
	require.NoError(t, LoadCacheFile(ctx, alz, path))
	require.NoError(t, alz.Init(ctx))

	out := filepath.Join(t.TempDir(), "saved.json.gz")
	require.NoError(t, SaveCacheFile(ctx, alz, out))
	assert.Equal(t, 0, cacheFileDefinitionCount(t, out))
}

// TestSaveBuiltInCacheAfterRead verifies that saving after the built-in definitions have been
// loaded, as the alz_architecture data source does, keeps them in the cache file.
func TestSaveBuiltInCacheAfterRead(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "alzlib-cache.json.gz")
	writeSeedCache(t, path)

	alz := newTestAlzLib(t)
	client := NewClient(WithAlzLib(alz), WithCacheFile(path, true))
	require.NoError(t, LoadCacheFile(ctx, alz, path))
	require.NoError(t, alz.Init(ctx))
	assert.Equal(t, 1, cacheFileDefinitionCount(t, path), "configuring must not overwrite the cache file")

	loadSeedDefinition(t, alz)
	require.NoError(t, client.SaveBuiltInCache(ctx))
	assert.Equal(t, 1, cacheFileDefinitionCount(t, path))
}

// TestSaveBuiltInCacheNoOp verifies that nothing is written unless a cache file is set and saving is enabled.
func TestSaveBuiltInCacheNoOp(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "alzlib-cache.json.gz")

	alz := newTestAlzLib(t)
	require.NoError(t, NewClient(WithAlzLib(alz), WithCacheFile(path, false)).SaveBuiltInCache(ctx))
	assert.NoFileExists(t, path)

	require.NoError(t, NewClient(WithAlzLib(alz), WithCacheFile("", true)).SaveBuiltInCache(ctx))
	require.NoError(t, NewClient(WithCacheFile(path, true)).SaveBuiltInCache(ctx))
	assert.NoFileExists(t, path)
}

// TestSaveBuiltInCacheConcurrent verifies that concurrent saves, as made by several alz_architecture
// data sources, all succeed and leave a valid cache file.
func TestSaveBuiltInCacheConcurrent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "alzlib-cache.json.gz")

	alz := newTestAlzLib(t)
	seed := filepath.Join(t.TempDir(), "seed.json.gz")
	writeSeedCache(t, seed)
	require.NoError(t, LoadCacheFile(ctx, alz, seed))
	loadSeedDefinition(t, alz)

	client := NewClient(WithAlzLib(alz), WithCacheFile(path, true))

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- client.SaveBuiltInCache(ctx)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, 1, cacheFileDefinitionCount(t, path))
}

func newTestAlzLib(t *testing.T) *alzlib.AlzLib {
	t.Helper()

	return alzlib.NewAlzLib(&alzlib.Options{Parallelism: 10})
}

// writeSeedCache writes a cache file containing one built-in policy definition.
func writeSeedCache(t *testing.T, path string) {
	t.Helper()

	desc := "test built-in policy definition"
	pdvs := assets.NewPolicyDefinitionVersions()
	require.NoError(t, pdvs.Add(&assets.PolicyDefinition{
		Definition: armpolicy.Definition{
			Name: to.Ptr(testBuiltInName),
			Properties: &armpolicy.DefinitionProperties{
				DisplayName: to.Ptr(testBuiltInName),
				Description: &desc,
				Metadata:    map[string]any{},
				PolicyRule:  map[string]any{"if": map[string]any{}, "then": map[string]any{}},
				PolicyType:  to.Ptr(armpolicy.PolicyTypeBuiltIn),
				Version:     to.Ptr("1.0.0"),
			},
		},
	}, false))

	c := cache.NewCacheFromDefinitions(
		map[string]*assets.PolicyDefinitionVersions{testBuiltInName: pdvs},
		map[string]*assets.PolicySetDefinitionVersions{},
	)
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, c.Save(f))
	require.NoError(t, f.Close())
}

// loadSeedDefinition loads the seed definition into the AlzLib from its cache, as reading an
// architecture does. No Azure call is made because the definition is in the cache.
func loadSeedDefinition(t *testing.T, alz *alzlib.AlzLib) {
	t.Helper()

	resID, err := arm.ParseResourceID("/providers/Microsoft.Authorization/policyDefinitions/" + testBuiltInName)
	require.NoError(t, err)
	require.NoError(t, alz.GetDefinitionsFromAzure(context.Background(), []alzlib.BuiltInRequest{
		{ResourceID: resID, Version: to.Ptr("1.0.*")},
	}))
}

func cacheFileDefinitionCount(t *testing.T, path string) int {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	c, err := cache.NewCache(f)
	require.NoError(t, err)

	return c.PolicyDefinitionCount() + c.PolicySetDefinitionCount()
}
