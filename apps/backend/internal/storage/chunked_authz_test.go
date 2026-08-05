package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// The chunked path must refuse to target a committed file (the overwrite-any-file
// bypass): only an in-progress upload, whose stored path is in the _chunks
// staging area, may be completed.
func TestAssertPendingUpload(t *testing.T) {
	svc, mock, tmpDir := setup(t)
	ctx := context.Background()

	// Committed file: path is a normal storage path (not under _chunks) -> forbidden.
	mock.ExpectQuery("SELECT path FROM files WHERE id").
		WithArgs("victimfile", "b", "p").
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(tmpDir + "/p/b/victimfile"))
	if err := svc.assertPendingUpload(ctx, "p", "b", "victimfile"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("committed file must be forbidden for chunk completion, got %v", err)
	}

	// Pending upload: path is under the _chunks staging root -> allowed.
	mock.ExpectQuery("SELECT path FROM files WHERE id").
		WithArgs("myupload", "b", "p").
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow(tmpDir + "/_chunks/myupload"))
	if err := svc.assertPendingUpload(ctx, "p", "b", "myupload"); err != nil {
		t.Fatalf("pending upload should be allowed, got %v", err)
	}

	// Unknown / other-project id: no row -> forbidden.
	mock.ExpectQuery("SELECT path FROM files WHERE id").
		WithArgs("nope", "b", "p").
		WillReturnError(errors.New("no rows"))
	if err := svc.assertPendingUpload(ctx, "p", "b", "nope"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("unknown id must be forbidden, got %v", err)
	}
}
