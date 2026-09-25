//go:build !windows

package config

import "os"

func mkdirPrivate(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

// writePrivate writes data to path with mode 0600. An existing file is
// narrowed to 0600 before any new content is written.
func writePrivate(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if err := f.Truncate(0); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
