package config

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// assertPrivate checks that path has a protected DACL whose only entry grants
// the current user access.
func assertPrivate(t *testing.T, path string, _ bool) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatalf("GetNamedSecurityInfo(%s) error = %v", path, err)
	}
	control, _, err := sd.Control()
	if err != nil {
		t.Fatalf("Control() error = %v", err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Errorf("%s DACL is not protected, parent entries are inherited", path)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("DACL() error = %v", err)
	}
	if dacl == nil {
		t.Fatalf("%s has a null DACL, which grants everyone access", path)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatalf("GetTokenUser() error = %v", err)
	}
	if dacl.AceCount != 1 {
		t.Errorf("%s DACL has %d entries, want 1: %s", path, dacl.AceCount, sd.String())
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			t.Fatalf("GetAce(%d) error = %v", i, err)
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || !sid.Equals(user.User.Sid) {
			t.Errorf("%s DACL entry %d is type %d for %s, want an allow entry for %s only",
				path, i, ace.Header.AceType, sid.String(), user.User.Sid.String())
		}
	}
}
