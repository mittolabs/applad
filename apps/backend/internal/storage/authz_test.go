package storage

import (
	"errors"
	"testing"

	"github.com/mittolabs/applad/internal/model"
)

func TestPermMatch(t *testing.T) {
	roles := storageRoles("abc") // any, users, user:abc
	if !permMatch([]string{`read("user:abc")`}, roles, "read") {
		t.Error("user:abc should match read(user:abc)")
	}
	if !permMatch([]string{"read(\"any\")"}, storageRoles(""), "read") {
		t.Error("anonymous should match read(any)")
	}
	if permMatch([]string{`read("user:xyz")`}, roles, "read") {
		t.Error("user:abc must not match read(user:xyz)")
	}
	if permMatch([]string{`read("users")`}, roles, "delete") {
		t.Error("read grant must not satisfy delete action")
	}
}

func TestAuthorizeFile(t *testing.T) {
	svc := &Service{}

	// Server API key (userID == "") always passes.
	if err := svc.authorizeFile(&model.Bucket{}, &model.File{}, "", "delete"); err != nil {
		t.Fatalf("API key should have full access: %v", err)
	}

	// Bucket-level read grants read to any file in the bucket.
	bucket := &model.Bucket{Permissions: []string{`read("user:abc")`}}
	if err := svc.authorizeFile(bucket, &model.File{}, "abc", "read"); err != nil {
		t.Fatalf("bucket read grant should allow: %v", err)
	}
	// A different user is denied.
	if err := svc.authorizeFile(bucket, &model.File{}, "xyz", "read"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other user should be forbidden, got %v", err)
	}

	// With file security OFF, a file-level permission does NOT grant access;
	// only the bucket governs.
	fsOff := &model.Bucket{FileSecurity: false}
	file := &model.File{Permissions: []string{`read("user:abc")`}}
	if err := svc.authorizeFile(fsOff, file, "abc", "read"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("file perm must not apply when file security is off, got %v", err)
	}

	// With file security ON, the file's own permission grants access.
	fsOn := &model.Bucket{FileSecurity: true}
	if err := svc.authorizeFile(fsOn, file, "abc", "read"); err != nil {
		t.Fatalf("file perm should apply with file security on: %v", err)
	}
	if err := svc.authorizeFile(fsOn, file, "abc", "delete"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("file read perm must not grant delete, got %v", err)
	}
}
