//go:build !windows

package config

import "os"

func mkdirPrivate(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

// createPrivateTemp creates a new file in dir with mode 0600.
func createPrivateTemp(dir, pattern string) (*os.File, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	// CreateTemp already uses 0600; state it rather than rely on that.
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	return f, nil
}
