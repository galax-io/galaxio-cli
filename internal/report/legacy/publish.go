package legacy

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Publish writes selected documents under runDir/js. Existing files require overwrite.
func Publish(runDir string, documents map[Product][]byte, overwrite bool) error {
	if len(documents) == 0 {
		return errors.New("no legacy report products selected")
	}

	jsDir := filepath.Join(runDir, "js")
	if err := prepareDirectory(jsDir); err != nil {
		return err
	}

	products := make([]Product, 0, len(documents))
	for _, product := range []Product{Stats, GlobalStats} {
		if _, selected := documents[product]; selected {
			products = append(products, product)
		}
	}
	if len(products) != len(documents) {
		return errors.New("invalid legacy report product")
	}

	for _, product := range products {
		path := filepath.Join(jsDir, product.Filename())
		info, err := os.Lstat(path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return fmt.Errorf("inspect %s: %w", path, err)
		case !overwrite:
			return fmt.Errorf("refuse to overwrite %s", path)
		case !info.Mode().IsRegular():
			return fmt.Errorf("refuse to replace non-file %s", path)
		}
	}

	for _, product := range products {
		path := filepath.Join(jsDir, product.Filename())
		err := createFile(path, documents[product])
		if overwrite && errors.Is(err, fs.ErrExist) {
			err = replaceFile(path, documents[product])
		}
		if err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	return nil
}

func prepareDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return os.Mkdir(path, 0o755)
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("refuse non-directory %s", path)
	}
	return nil
}

func createFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = io.ErrShortWrite
	}
	err = errors.Join(writeErr, file.Close())
	if err != nil {
		err = errors.Join(err, os.Remove(path))
	}
	return err
}

// replaceFile keeps the original intact until the replacement is completely written.
func replaceFile(path string, data []byte) (err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refuse to replace non-file %s", path)
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(file.Name()))
		}
	}()
	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = io.ErrShortWrite
	}
	if err = errors.Join(writeErr, file.Chmod(info.Mode().Perm()), file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
