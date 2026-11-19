// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package clients

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Azure/alzlib"
	"github.com/Azure/alzlib/cache"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// LoadCacheFile loads the gzipped cache file at the given path and injects it
// into the AlzLib. If the file does not exist, this is treated as a no-op so
// that the cache file can be created on first run when used with
// `cache_file_save_enabled = true`.
func LoadCacheFile(ctx context.Context, alz *alzlib.AlzLib, path string) error {
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

	c, err := cache.NewCache(f)
	if err != nil {
		return fmt.Errorf("reading cache file %q: %w", path, err)
	}
	alz.AddCache(c)
	tflog.Debug(ctx, "Loaded AlzLib built-in cache from file", map[string]interface{}{
		"cache_file_name": path,
	})
	return nil
}

// SaveCacheFile exports the built-in policy and policy set definitions from the
// AlzLib and writes them to the given path as a gzipped JSON file. The write is
// performed via a temporary file in the same directory and renamed atomically
// to avoid leaving a corrupt cache file if the process is interrupted.
//
// Only definitions that have been loaded into the AlzLib are exported. A cache
// added with AddCache is read lazily, so built-in definitions are only present
// after an architecture has been built, e.g. by the alz_architecture data source.
func SaveCacheFile(ctx context.Context, alz *alzlib.AlzLib, path string) error {
	c := alz.ExportBuiltInCache()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary cache file in %q: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we don't make it to the rename.
	defer func() { _ = os.Remove(tmpName) }()

	if err := c.Save(tmp); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing cache file %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing cache file %q: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		// On some platforms (notably older Windows behaviour) os.Rename can
		// fail when the destination file already exists. Fall back to
		// removing the existing destination and renaming again so
		// cache_file_save_enabled works on the second and subsequent runs.
		// The temp file is fully written and closed before this point, so
		// the file content remains valid.
		if _, statErr := os.Stat(path); statErr != nil {
			return fmt.Errorf("renaming cache file %q to %q: %w", tmpName, path, err)
		}
		if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
			return fmt.Errorf("removing existing cache file %q: %w", path, rmErr)
		}
		if err := os.Rename(tmpName, path); err != nil {
			return fmt.Errorf("renaming cache file %q to %q: %w", tmpName, path, err)
		}
	}
	tflog.Debug(ctx, "Saved AlzLib built-in cache to file", map[string]interface{}{
		"cache_file_name":        path,
		"policy_definitions":     c.PolicyDefinitionCount(),
		"policy_set_definitions": c.PolicySetDefinitionCount(),
	})
	return nil
}
