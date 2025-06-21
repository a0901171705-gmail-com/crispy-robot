package filepaths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MissingStrategy interface {
	missingStrategySealed()
}

type missingNotOk struct{}

func (missingNotOk) missingStrategySealed() {}

type missingOk struct{}

func (missingOk) missingStrategySealed() {}

var (
	MissingOk    MissingStrategy = missingOk{}
	MissingNotOk MissingStrategy = missingNotOk{}
)

func ParseMissingOk(missingOk bool) MissingStrategy {
	if missingOk {
		return MissingOk
	}
	return MissingNotOk
}

// Might be unnecessary type, I dunno. Might be useful to have the path.
type CanonicalisationError struct {
	AttemptedPath   string
	UnderlyingError error
}

func (e CanonicalisationError) Error() string {
	return fmt.Sprintf("could not canonicalise path %q due to %v", e.AttemptedPath, e.UnderlyingError)
}

func Canonicalise(path string, missingStrategy MissingStrategy) (Canonicalised, error) {
	if missingStrategy == nil {
		panic("missingStrategy must not be nil")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, CanonicalisationError{AttemptedPath: path, UnderlyingError: err}
	}

	symlinkEvaledPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if missingStrategy == MissingOk && os.IsNotExist(err) {
			return canonicalised{path: absPath}, nil
		}

		return nil, CanonicalisationError{AttemptedPath: absPath, UnderlyingError: err}
	}

	return canonicalised{path: symlinkEvaledPath}, nil
}

type Canonicalised interface {
	canonicalisedSealed()
	String() string
	IsAncestorOf(Canonicalised) (bool, error)
	Join(string, MissingStrategy) (Canonicalised, error)
}

type canonicalised struct {
	path string
}

func (c canonicalised) canonicalisedSealed() {}

func (c canonicalised) String() string {
	return c.path
}

func (c canonicalised) IsAncestorOf(other Canonicalised) (bool, error) {
	relativePath, err := filepath.Rel(c.path, other.String())
	if err != nil {
		// TODO: maybe wrap this, not sure
		return false, err
	}
	return !strings.HasPrefix(relativePath, ".."), nil
}

func (c canonicalised) Join(path string, missingStrategy MissingStrategy) (Canonicalised, error) {
	return Canonicalise(filepath.Join(c.path, path), missingStrategy)
}
