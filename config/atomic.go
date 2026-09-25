package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// writeData writes data to the temporary config file. Tests replace it to
// simulate a failed write.
var writeData = func(f *os.File, data []byte) error {
	_, err := f.Write(data)
	return err
}

// writePrivate replaces path with data, restricted to the current user. The
// data goes to a temporary file in the same directory, which is renamed over
// path only once it is fully written, so a failed write leaves the previous
// file intact. A symlinked path keeps the link and replaces its target.
func writePrivate(path string, data []byte) error {
	path, err := resolveTarget(path)
	if err != nil {
		return err
	}
	f, err := createPrivateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if err := writeData(f, data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// resolveTarget follows symlinks so the rename replaces the file the link
// points to rather than the link itself. A missing path is returned as is.
func resolveTarget(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		return path, nil
	}
	return filepath.EvalSymlinks(path)
}
