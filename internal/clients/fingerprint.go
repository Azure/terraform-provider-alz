// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package clients

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Azure/alzlib"
)

// LibraryFingerprint returns a fingerprint of the libraries, in order, and of the library overwrite
// setting, which together determine the built-in definitions that an architecture needs.
// ALZ Library references are identified by path and tag, because released tags are not moved.
// For other libraries, the files that alzlib reads are hashed, so that a change to a local
// library changes the fingerprint, while its location, which can differ between machines, does not.
// The libraries must have been fetched.
func LibraryFingerprint(refs alzlib.LibraryReferences, libraryOverwriteEnabled bool) (string, error) {
	h := sha256.New()
	writeLine(h, "library_overwrite_enabled %t", libraryOverwriteEnabled)

	alzlibDir := alzlibDirBase()
	for i, ref := range refs {
		if alzRef, ok := ref.(*alzlib.AlzLibraryReference); ok {
			writeLine(h, "%d alz %q", i, alzRef.String())
			continue
		}

		fsys := ref.FS()
		if fsys == nil {
			return "", fmt.Errorf("library %s has not been fetched", ref.String())
		}

		writeLine(h, "%d custom", i)
		err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !isLibraryFile(name, alzlibDir) {
				return err
			}
			content, err := fs.ReadFile(fsys, name)
			if err != nil {
				return err
			}
			writeLine(h, "%q %x", name, sha256.Sum256(content))
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("fingerprinting library %s: %w", ref.String(), err)
		}
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// isLibraryFile reports whether alzlib reads the file when it processes a library, mirroring
// the alzlib processor: JSON and YAML files, except those whose path contains the base name of
// the directory that alzlib fetches libraries to. Other files, such as the .git directory of a
// library fetched with git, would make the fingerprint differ between runs.
func isLibraryFile(name, alzlibDir string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".json", ".yaml", ".yml":
		return !strings.Contains(name, alzlibDir)
	default:
		return false
	}
}

// alzlibDirBase returns the base name of the directory that alzlib fetches libraries to: the
// ALZLIB_DIR environment variable, or .alzlib by default.
func alzlibDirBase() string {
	dir := ".alzlib"
	if d := os.Getenv("ALZLIB_DIR"); d != "" {
		dir = d
	}
	return filepath.Base(dir)
}

// writeLine writes a formatted line to w, which must not fail, such as a hash.
func writeLine(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", a...)
}
