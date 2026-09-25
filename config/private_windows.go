package config

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// os.WriteFile and os.MkdirAll ignore the Unix mode on Windows, so new files
// inherit the parent's ACL. These helpers apply a protected DACL with a single
// full-control entry for the current user instead.

// currentUserSDDL returns an SDDL DACL that grants the current user full
// control, does not inherit from the parent, and passes the given inheritance
// flags (e.g. "OICI" for directories) to children.
func currentUserSDDL(inherit string) (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("get current user: %w", err)
	}
	return fmt.Sprintf("D:P(A;%s;FA;;;%s)", inherit, user.User.Sid.String()), nil
}

func privateSecurityAttributes(inherit string) (*windows.SecurityAttributes, *windows.SECURITY_DESCRIPTOR, error) {
	sddl, err := currentUserSDDL(inherit)
	if err != nil {
		return nil, nil, err
	}
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return nil, nil, fmt.Errorf("build security descriptor: %w", err)
	}
	sa := &windows.SecurityAttributes{SecurityDescriptor: sd}
	sa.Length = uint32(unsafe.Sizeof(*sa))
	return sa, sd, nil
}

// mkdirPrivate creates dir restricted to the current user. Missing parents
// are created normally, and an existing dir is left alone: it may be a
// shared location the user pointed --config at.
func mkdirPrivate(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return err
	}
	sa, _, err := privateSecurityAttributes("OICI")
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	if err := windows.CreateDirectory(p, sa); err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return &os.PathError{Op: "mkdir", Path: dir, Err: err}
	}
	return nil
}

// createPrivateTemp creates a new file in dir restricted to the current user.
// Renaming it within the volume keeps that ACL.
func createPrivateTemp(dir, pattern string) (*os.File, error) {
	sa, _, err := privateSecurityAttributes("")
	if err != nil {
		return nil, err
	}
	for range 10000 {
		name := filepath.Join(dir, pattern+rand.Text())
		p, err := windows.UTF16PtrFromString(name)
		if err != nil {
			return nil, err
		}
		h, err := windows.CreateFile(p,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0, sa, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
		if errors.Is(err, windows.ERROR_FILE_EXISTS) {
			continue
		}
		if err != nil {
			return nil, &os.PathError{Op: "createtemp", Path: name, Err: err}
		}
		return os.NewFile(uintptr(h), name), nil
	}
	return nil, &os.PathError{Op: "createtemp", Path: filepath.Join(dir, pattern+"*"), Err: os.ErrExist}
}
