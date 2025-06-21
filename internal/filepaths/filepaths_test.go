package filepaths_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cli/cli/v2/internal/filepaths"
	"github.com/stretchr/testify/require"
)

func TestCanonicalisationHappyPath(t *testing.T) {
	tempDir := canonicalisedTempDir(t).String()
	workingDir := filepath.Join(tempDir, "working")
	require.NoError(t, os.Mkdir(workingDir, 0o700))
	require.NoError(t, os.Chdir(workingDir))

	tests := map[string]struct {
		setup        func(t *testing.T)
		input        string
		expectedPath string
		expectedErr  error
	}{
		"root": {
			input:        "/",
			expectedPath: "/",
		},
		"non-root": {
			input:        tempDir,
			expectedPath: tempDir,
		},
		"current directory": {
			input:        ".",
			expectedPath: workingDir,
		},
		"parent directory": {
			input:        "..",
			expectedPath: tempDir,
		},
		"symlinks resolved": {
			setup: func(t *testing.T) {
				targetDir := filepath.Join(tempDir, "target")
				require.NoError(t, os.Mkdir(targetDir, 0o700))

				symlink := filepath.Join(workingDir, "symlink")
				require.NoError(t, os.Symlink(targetDir, symlink))
			},
			input:        filepath.Join(workingDir, "symlink"),
			expectedPath: filepath.Join(tempDir, "target"),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.setup != nil {
				tc.setup(t)
			}

			canonicalisedPath, err := filepaths.Canonicalise(tc.input, filepaths.MissingOk)
			require.NoError(t, err)
			require.Equal(t, tc.expectedPath, canonicalisedPath.String())
		})
	}
}

func TestCanonicalisationMissingPaths(t *testing.T) {
	// When missing is not ok, returns an error
	var error filepaths.CanonicalisationError
	_, err := filepaths.Canonicalise("/presumably/nonexistent/path", filepaths.MissingNotOk)
	require.ErrorAs(t, err, &error)

	require.Equal(t, "/presumably/nonexistent/path", error.AttemptedPath)

	// When missing is ok, returns the path
	canonicalisedPath, err := filepaths.Canonicalise("/presumably/nonexistent/path", filepaths.MissingOk)
	require.NoError(t, err)
	require.Equal(t, "/presumably/nonexistent/path", canonicalisedPath.String())
}

func canonicalisedTempDir(t *testing.T) filepaths.Canonicalised {
	t.Helper()

	tempDir := t.TempDir()
	canonicalisedPath, err := filepaths.Canonicalise(tempDir, filepaths.MissingOk)
	require.NoError(t, err)

	return canonicalisedPath
}
