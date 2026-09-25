package config

import (
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

// writePrivate writes data to path restricted to the current user. A new file
// is created with the restricted ACL, and an existing file gets it before
// any new content is written.
func writePrivate(path string, data []byte) error {
	sa, sd, err := privateSecurityAttributes("")
	if err != nil {
		return err
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p,
		windows.GENERIC_WRITE|windows.WRITE_DAC,
		0, sa, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return &os.PathError{Op: "open", Path: path, Err: err}
	}
	f := os.NewFile(uintptr(h), path)

	// OPEN_ALWAYS ignores the security attributes for an existing file.
	dacl, _, err := sd.DACL()
	if err == nil {
		err = windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT,
			windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
			nil, nil, dacl, nil)
	}
	if err != nil {
		f.Close()
		return &os.PathError{Op: "set acl", Path: path, Err: err}
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
