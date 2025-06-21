package download

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirMode  os.FileMode = 0755
	fileMode os.FileMode = 0644
	execMode os.FileMode = 0755
)

func extractZip(zr *zip.Reader, destDir string) error {
	for _, zf := range zr.File {
		fpath := filepath.Join(destDir, filepath.FromSlash(zf.Name))
		if !filepathDescendsFrom(fpath, destDir) {
			continue
		}
		if err := extractZipFile(zf, fpath); err != nil {
			return fmt.Errorf("error extracting %q: %w", zf.Name, err)
		}
	}
	return nil
}

func extractZipFile(zf *zip.File, dest string) (extractErr error) {
	zm := zf.Mode()
	if zm.IsDir() {
		extractErr = os.MkdirAll(dest, dirMode)
		return
	}

	var f io.ReadCloser
	f, extractErr = zf.Open()
	if extractErr != nil {
		return
	}
	defer f.Close()

	if dir := filepath.Dir(dest); dir != "." {
		if extractErr = os.MkdirAll(dir, dirMode); extractErr != nil {
			return
		}
	}

	var df *os.File
	if df, extractErr = os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, getPerm(zm)); extractErr != nil {
		return
	}

	defer func() {
		if err := df.Close(); extractErr == nil && err != nil {
			extractErr = err
		}
	}()

	_, extractErr = io.Copy(df, f)
	return
}

func getPerm(m os.FileMode) os.FileMode {
	if m&0111 == 0 {
		return fileMode
	}
	return execMode
}

func filepathDescendsFrom(p, dir string) bool {
	// Regardless of the logic below, `p` is never allowed to be current directory `.` or parent directory `..`
	// however we check explicitly here before filepath.Rel() which doesn't cover all cases.
	p = filepath.Clean(p)

	if p == "." || p == ".." {
		return false
	}

	// filepathDescendsFrom() takes advantage of filepath.Rel() to determine if `p` is descended from `dir`:
	//
	// 1. filepath.Rel() calculates a path to traversal from fictious `dir` to `p`.
	// 2. filepath.Rel() errors in a handful of cases where absolute and relative paths are compared as well as certain traversal edge cases
	//    For more information, https://github.com/golang/go/blob/00709919d09904b17cfe3bfeb35521cbd3fb04f8/src/path/filepath/path_test.go#L1510-L1515
	// 3. If the path to traverse `dir` to `p` requires `..`, then we know it is not descend from / contained in `dir`
	//
	// As-is, this function requires the caller to ensure `p` and `dir` are either 1) both relative or 2) both absolute.
	relativePath, err := filepath.Rel(dir, p)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(relativePath, "..")
}
