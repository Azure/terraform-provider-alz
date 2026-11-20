// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package clients

import (
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/Azure/alzlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibraryFingerprint(t *testing.T) {
	alzFS := fstest.MapFS{"policy.json": {Data: []byte("a")}}
	localFS := fstest.MapFS{"archetype.json": {Data: []byte("x")}}

	fingerprint := func(overwrite bool, refs ...alzlib.LibraryReference) string {
		t.Helper()
		fp, err := LibraryFingerprint(refs, overwrite)
		require.NoError(t, err)
		return fp
	}

	alzRef := alzlib.NewAlzLibraryReferenceFromFS("platform/alz", "2026.11.0", alzFS)
	localRef := alzlib.NewCustomLibraryReferenceFromFS("/agent-1/lib", localFS)
	base := fingerprint(false, alzRef, localRef)

	assert.Equal(t, base, fingerprint(false, alzRef, localRef), "the fingerprint must be stable")
	assert.Equal(t, base, fingerprint(false,
		alzlib.NewAlzLibraryReferenceFromFS("platform/alz", "2026.11.0", fstest.MapFS{}),
		localRef,
	), "an ALZ Library reference is identified by path and tag")
	assert.Equal(t, base, fingerprint(false,
		alzRef,
		alzlib.NewCustomLibraryReferenceFromFS("/agent-2/lib", localFS),
	), "the location of a custom library does not matter")

	assert.NotEqual(t, base, fingerprint(false,
		alzlib.NewAlzLibraryReferenceFromFS("platform/alz", "2026.12.0", alzFS),
		localRef,
	), "a different tag")
	assert.NotEqual(t, base, fingerprint(false,
		alzRef,
		alzlib.NewCustomLibraryReferenceFromFS("/agent-1/lib", fstest.MapFS{"archetype.json": {Data: []byte("y")}}),
	), "a changed custom library")
	assert.NotEqual(t, base, fingerprint(false, localRef, alzRef), "a different order")
	assert.NotEqual(t, base, fingerprint(true, alzRef, localRef), "library_overwrite_enabled")

	withLocalFiles := func(files fstest.MapFS) alzlib.LibraryReference {
		files["archetype.json"] = localFS["archetype.json"]
		return alzlib.NewCustomLibraryReferenceFromFS("/agent-1/lib", files)
	}
	assert.Equal(t, base, fingerprint(false, alzRef, withLocalFiles(fstest.MapFS{
		"README.md":                   {Data: []byte("readme")},
		".git/index":                  {Data: []byte("differs between clones")},
		".alzlib/123/lib/policy.json": {Data: []byte("a library fetched by alzlib")},
	})), "files that alzlib does not read")
	assert.NotEqual(t, base, fingerprint(false, alzRef, withLocalFiles(fstest.MapFS{
		"test.alz_policy_assignment.YAML": {Data: []byte("y")},
	})), "a new library file")

	t.Setenv("ALZLIB_DIR", filepath.Join(t.TempDir(), "alzlib-dir"))
	assert.Equal(t, base, fingerprint(false, alzRef, withLocalFiles(fstest.MapFS{
		"alzlib-dir/123/lib/policy.json": {Data: []byte("a library fetched by alzlib")},
	})), "files in ALZLIB_DIR")

	_, err := LibraryFingerprint(alzlib.LibraryReferences{alzlib.NewCustomLibraryReference("./lib")}, false)
	assert.ErrorContains(t, err, "has not been fetched")
}
