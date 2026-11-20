// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/alzlib"
	"github.com/Azure/alzlib/cache"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// cacheFileMetadataSchemaVersion is the version of the cacheFileMetadata format.
const cacheFileMetadataSchemaVersion = 1

// cacheFileMetadata is saved by the provider next to the cache file. A cache file saved by the
// provider only holds the versions of the built-in definitions that its libraries needed, so the
// metadata records which libraries those were, see LibraryFingerprint.
type cacheFileMetadata struct {
	SchemaVersion      int    `json:"schemaVersion"`
	LibraryFingerprint string `json:"libraryFingerprint"`
}

// CacheFileMetadataPath returns the path of the metadata file that is saved next to the cache file.
func CacheFileMetadataPath(path string) string {
	return path + ".meta.json"
}

// LoadCacheFile loads the gzipped cache file at the given path and injects it
// into the AlzLib. If the file does not exist, this is treated as a no-op so
// that the cache file can be created on first run when used with
// `cache_file_save_enabled = true`.
// A cache file that the provider saved for other libraries is not loaded. A cache file without
// metadata, e.g. one created by alzlibtool, is always loaded.
func LoadCacheFile(ctx context.Context, alz *alzlib.AlzLib, path, libraryFingerprint string) error {
	start := time.Now()
	f, err := os.Open(path) // #nosec G304 -- path is provided by the operator via provider config.
	if err != nil {
		if os.IsNotExist(err) {
			tflog.Debug(ctx, "Cache file does not exist, skipping load", map[string]interface{}{
				"cache_file_name": path,
			})
			return nil
		}
		return fmt.Errorf("opening cache file %q: %w", path, err)
	}
	defer f.Close() // #nosec G307 -- read-only.

	if ok, reason := cacheFileMatchesLibraries(path, libraryFingerprint); !ok {
		tflog.Info(ctx, "Cache file was not saved for these libraries, skipping load", map[string]interface{}{
			"cache_file_name": path,
			"reason":          reason,
		})
		return nil
	}

	c, err := cache.NewCache(f)
	if err != nil {
		return fmt.Errorf("reading cache file %q: %w", path, err)
	}
	alz.AddCache(c)
	tflog.Debug(ctx, "Loaded AlzLib built-in cache from file", map[string]interface{}{
		"cache_file_name":        path,
		"policy_definitions":     c.PolicyDefinitionCount(),
		"policy_set_definitions": c.PolicySetDefinitionCount(),
		"duration_ms":            time.Since(start).Milliseconds(),
	})
	return nil
}

// cacheFileMatchesLibraries reports whether the cache file can be used with the libraries that
// have the given fingerprint, and if not, why.
func cacheFileMatchesLibraries(path, libraryFingerprint string) (bool, string) {
	content, err := os.ReadFile(CacheFileMetadataPath(path)) // #nosec G304 -- next to the cache file provided by the operator.
	if errors.Is(err, os.ErrNotExist) {
		return true, ""
	}
	if err != nil {
		return false, fmt.Sprintf("reading metadata file: %v", err)
	}

	var meta cacheFileMetadata
	if err := json.Unmarshal(content, &meta); err != nil {
		return false, fmt.Sprintf("parsing metadata file: %v", err)
	}
	switch {
	case meta.SchemaVersion != cacheFileMetadataSchemaVersion:
		return false, fmt.Sprintf("unsupported metadata schema version %d", meta.SchemaVersion)
	case meta.LibraryFingerprint == "":
		return false, "the cache file was not saved completely"
	case meta.LibraryFingerprint != libraryFingerprint:
		return false, "the libraries have changed"
	}
	return true, ""
}

// SaveCacheFile exports the built-in policy and policy set definitions from the
// AlzLib and writes them to the given path as a gzipped JSON file. The write is
// performed via a temporary file in the same directory and renamed atomically
// to avoid leaving a corrupt cache file if the process is interrupted.
//
// Only definitions that have been loaded into the AlzLib are exported. A cache
// added with AddCache is read lazily, so built-in definitions are only present
// after an architecture has been built, e.g. by the alz_architecture data source.
//
// If libraryFingerprint is set, it is saved in the metadata file next to the cache file, so that
// LoadCacheFile does not use the cache file for other libraries.
func SaveCacheFile(ctx context.Context, alz *alzlib.AlzLib, path, libraryFingerprint string) error {
	start := time.Now()
	c := alz.ExportBuiltInCache()

	if libraryFingerprint != "" {
		// Invalidate the metadata first, so that the cache file is not used if saving stops halfway.
		if err := writeCacheFileMetadata(path, ""); err != nil {
			return err
		}
	}

	if err := writeFileAtomically(path, c.Save); err != nil {
		return fmt.Errorf("writing cache file %q: %w", path, err)
	}

	if libraryFingerprint != "" {
		if err := writeCacheFileMetadata(path, libraryFingerprint); err != nil {
			return err
		}
	}

	tflog.Debug(ctx, "Saved AlzLib built-in cache to file", map[string]interface{}{
		"cache_file_name":        path,
		"policy_definitions":     c.PolicyDefinitionCount(),
		"policy_set_definitions": c.PolicySetDefinitionCount(),
		"duration_ms":            time.Since(start).Milliseconds(),
	})
	return nil
}

func writeCacheFileMetadata(path, libraryFingerprint string) error {
	meta := cacheFileMetadata{
		SchemaVersion:      cacheFileMetadataSchemaVersion,
		LibraryFingerprint: libraryFingerprint,
	}
	metaPath := CacheFileMetadataPath(path)
	if err := writeFileAtomically(metaPath, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(meta)
	}); err != nil {
		return fmt.Errorf("writing cache metadata file %q: %w", metaPath, err)
	}
	return nil
}

// writeFileAtomically writes a file via a temporary file in the same directory, which is then
// renamed, so that an interrupted write never leaves a partial file.
func writeFileAtomically(path string, write func(io.Writer) error) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary file in %q: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we don't make it to the rename.
	defer func() { _ = os.Remove(tmpName) }()

	if err := write(tmp); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing %q: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		// On some platforms (notably older Windows behaviour) os.Rename can
		// fail when the destination file already exists. Fall back to
		// removing the existing destination and renaming again so
		// cache_file_save_enabled works on the second and subsequent runs.
		// The temp file is fully written and closed before this point, so
		// the file content remains valid.
		if _, statErr := os.Stat(path); statErr != nil {
			return fmt.Errorf("renaming %q to %q: %w", tmpName, path, err)
		}
		if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("removing existing %q: %w", path, rmErr)
		}
		if err := os.Rename(tmpName, path); err != nil {
			return fmt.Errorf("renaming %q to %q: %w", tmpName, path, err)
		}
	}
	return nil
}
