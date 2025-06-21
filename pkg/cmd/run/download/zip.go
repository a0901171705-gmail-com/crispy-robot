package download

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cli/cli/v2/internal/filepaths"
)

const (
	dirMode  os.FileMode = 0755
	fileMode os.FileMode = 0644
	execMode os.FileMode = 0755
)

func extractZip(zr *zip.Reader, destDir filepaths.Canonicalised) error {
	for _, zf := range zr.File {
		fpath, err := destDir.Join(filepath.FromSlash(zf.Name), filepaths.MissingOk)
		if err != nil {
			// TODO: consider error
			return err
		}
		contains, err := destDir.IsAncestorOf(fpath)
		if err != nil {
			// TODO: consider error
			return err
		}
		if !contains {
			// TODO: probably error
			continue
		}
		if err := extractZipFile(zf, fpath); err != nil {
			return fmt.Errorf("error extracting %q: %w", zf.Name, err)
		}
	}
	return nil
}

func extractZipFile(zf *zip.File, dest filepaths.Canonicalised) (extractErr error) {
	zm := zf.Mode()
	if zm.IsDir() {
		extractErr = os.MkdirAll(dest.String(), dirMode)
		return
	}

	var f io.ReadCloser
	f, extractErr = zf.Open()
	if extractErr != nil {
		return
	}
	defer f.Close()

	// TODO: not sure exactly about this logic tbh, since we might have absolute paths at this point.
	if dir := filepath.Dir(dest.String()); dir != "." {
		if extractErr = os.MkdirAll(dir, dirMode); extractErr != nil {
			return
		}
	}

	var df *os.File
	if df, extractErr = os.OpenFile(dest.String(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, getPerm(zm)); extractErr != nil {
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

// TODO: This isn't actually used anywhere, I just wanted to use the tests
// and see if they worked for Canonicalise.
func filepathDescendsFrom(p, dir string) bool {
	cP, err := filepaths.Canonicalise(p, filepaths.MissingOk)
	mustNot(err)
	cDir, err := filepaths.Canonicalise(dir, filepaths.MissingOk)
	mustNot(err)

	contains, err := cDir.IsAncestorOf(cP)
	mustNot(err)
	return contains
}

func mustNot(err error) {
	if err != nil {
		panic(err)
	}
}
