// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package clients

import (
	"context"
	"encoding/json"
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

const (
	testBuiltInName   = "00000000-0000-0000-0000-000000000001"
	testFingerprint   = "sha256:test"
	otherFingerprint  = "sha256:other"
	testCacheFileName = "alzlib-cache.json.gz"
)

// TestSaveAndLoadCacheFile verifies that the cache file helpers can round-trip
// an empty AlzLib's built-in cache through a gzipped file on disk.
func TestSaveAndLoadCacheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), testCacheFileName)

	alz := alzlib.NewAlzLib(nil)
	ctx := context.Background()

	// Saving with no built-ins loaded should still produce a valid cache file.
	require.NoError(t, SaveCacheFile(ctx, alz, path, ""))
	assert.FileExists(t, path)
	assert.NoFileExists(t, CacheFileMetadataPath(path), "no metadata is written without a fingerprint")

	// Loading the file we just wrote should succeed and inject a cache.
	alz2 := alzlib.NewAlzLib(nil)
	require.NoError(t, LoadCacheFile(ctx, alz2, path, ""))
}

// TestLoadCacheFileMissingIsNoOp verifies that loading a non-existent cache file
// is treated as a no-op so that the cache file can be created on first run when
// `cache_file_save_enabled` is true.
func TestLoadCacheFileMissingIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json.gz")

	alz := alzlib.NewAlzLib(nil)
	require.NoError(t, LoadCacheFile(context.Background(), alz, path, testFingerprint))
}

// TestLoadCacheFileInvalid verifies that a non-gzip file produces an error.
func TestLoadCacheFileInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.gz")
	// Write some non-gzip data.
	require.NoError(t, os.WriteFile(path, []byte("not-a-gzip-file"), 0o600))

	alz := alzlib.NewAlzLib(nil)
	err := LoadCacheFile(context.Background(), alz, path, testFingerprint)
	assert.Error(t, err)
}

// TestSaveCacheFileOverwrite verifies that SaveCacheFile can be called more
// than once against the same destination path. This guards against platform
// differences (notably Windows) where os.Rename can fail if the destination
// already exists.
func TestSaveCacheFileOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), testCacheFileName)

	alz := alzlib.NewAlzLib(nil)
	ctx := context.Background()

	require.NoError(t, SaveCacheFile(ctx, alz, path, testFingerprint))
	assert.FileExists(t, path)

	// A second save against the same path must succeed (overwrite).
	require.NoError(t, SaveCacheFile(ctx, alz, path, testFingerprint))
	assert.FileExists(t, path)

	// And the file should still be loadable after overwrite.
	alz2 := alzlib.NewAlzLib(nil)
	require.NoError(t, LoadCacheFile(ctx, alz2, path, testFingerprint))
}

// TestSaveCacheFileExportsOnlyLoadedDefinitions documents why the cache cannot be saved when the
// provider is configured: a loaded cache is read lazily, so right after Init the AlzLib holds no
// built-in definitions and saving would overwrite a good cache with an empty one
// (Azure/Azure-Landing-Zones#4328).
func TestSaveCacheFileExportsOnlyLoadedDefinitions(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), testCacheFileName)
	writeSeedCache(t, path)

	alz := newTestAlzLib(t)
	require.NoError(t, LoadCacheFile(ctx, alz, path, testFingerprint))
	require.NoError(t, alz.Init(ctx))

	out := filepath.Join(t.TempDir(), "saved.json.gz")
	require.NoError(t, SaveCacheFile(ctx, alz, out, testFingerprint))
	assert.Equal(t, 0, cacheFileDefinitionCount(t, out))
}

// TestSaveBuiltInCacheAfterRead verifies that saving after the built-in definitions have been
// loaded, as the alz_architecture data source does, keeps them in the cache file.
func TestSaveBuiltInCacheAfterRead(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), testCacheFileName)
	writeSeedCache(t, path)

	alz := newTestAlzLib(t)
	client := NewClient(WithAlzLib(alz), WithCacheFile(path, true, testFingerprint))
	require.NoError(t, alz.Init(ctx))
	require.NoError(t, LoadCacheFile(ctx, alz, path, testFingerprint))
	assert.Equal(t, 1, cacheFileDefinitionCount(t, path), "configuring must not overwrite the cache file")

	require.NoError(t, loadSeedDefinition(alz))
	require.NoError(t, client.SaveBuiltInCache(ctx))
	assert.Equal(t, 1, cacheFileDefinitionCount(t, path))
	assert.Equal(t, testFingerprint, readCacheFileMetadata(t, path).LibraryFingerprint)
}

// TestSaveBuiltInCacheNoOp verifies that nothing is written unless a cache file is set and saving is enabled.
func TestSaveBuiltInCacheNoOp(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), testCacheFileName)

	alz := newTestAlzLib(t)
	require.NoError(t, NewClient(WithAlzLib(alz), WithCacheFile(path, false, testFingerprint)).SaveBuiltInCache(ctx))
	assert.NoFileExists(t, path)

	require.NoError(t, NewClient(WithAlzLib(alz), WithCacheFile("", true, testFingerprint)).SaveBuiltInCache(ctx))
	require.NoError(t, NewClient(WithCacheFile(path, true, testFingerprint)).SaveBuiltInCache(ctx))
	assert.NoFileExists(t, path)
}

// TestSaveBuiltInCacheConcurrent verifies that concurrent saves, as made by several alz_architecture
// data sources, all succeed and leave a valid cache file.
func TestSaveBuiltInCacheConcurrent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), testCacheFileName)

	alz := newTestAlzLib(t)
	seed := filepath.Join(t.TempDir(), "seed.json.gz")
	writeSeedCache(t, seed)
	require.NoError(t, LoadCacheFile(ctx, alz, seed, testFingerprint))
	require.NoError(t, loadSeedDefinition(alz))

	client := NewClient(WithAlzLib(alz), WithCacheFile(path, true, testFingerprint))

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
	assert.Equal(t, testFingerprint, readCacheFileMetadata(t, path).LibraryFingerprint)
}

// TestLoadCacheFileMetadata verifies that a cache file is only loaded for the libraries it was saved
// for, and that a cache file without metadata, e.g. created by alzlibtool, is always loaded.
func TestLoadCacheFileMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		loaded   bool
	}{
		{name: "no metadata", loaded: true},
		{name: "same libraries", metadata: `{"schemaVersion":1,"libraryFingerprint":"` + testFingerprint + `"}`, loaded: true},
		{name: "other libraries", metadata: `{"schemaVersion":1,"libraryFingerprint":"` + otherFingerprint + `"}`},
		{name: "saving stopped halfway", metadata: `{"schemaVersion":1,"libraryFingerprint":""}`},
		{name: "unknown schema version", metadata: `{"schemaVersion":2,"libraryFingerprint":"` + testFingerprint + `"}`},
		{name: "corrupt metadata", metadata: `{`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), testCacheFileName)
			writeSeedCache(t, path)
			if tc.metadata != "" {
				require.NoError(t, os.WriteFile(CacheFileMetadataPath(path), []byte(tc.metadata), 0o600))
			}

			alz := newTestAlzLib(t)
			require.NoError(t, LoadCacheFile(context.Background(), alz, path, testFingerprint))

			// Without a cache the definition has to be fetched from Azure, which fails as no policy client is set.
			err := loadSeedDefinition(alz)
			if tc.loaded {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, "policy client not set")
			}
		})
	}
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

// loadSeedDefinition loads the seed definition into the AlzLib, as reading an architecture does.
// It is served from the cache if one is loaded; otherwise it would be fetched from Azure.
func loadSeedDefinition(alz *alzlib.AlzLib) error {
	resID, err := arm.ParseResourceID("/providers/Microsoft.Authorization/policyDefinitions/" + testBuiltInName)
	if err != nil {
		return err
	}

	return alz.GetDefinitionsFromAzure(context.Background(), []alzlib.BuiltInRequest{
		{ResourceID: resID, Version: to.Ptr("1.0.*")},
	})
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

func readCacheFileMetadata(t *testing.T, path string) cacheFileMetadata {
	t.Helper()

	content, err := os.ReadFile(CacheFileMetadataPath(path))
	require.NoError(t, err)

	var meta cacheFileMetadata
	require.NoError(t, json.Unmarshal(content, &meta))

	return meta
}
