package transfer

import (
	"context"
	"encoding/json"

	"github.com/mittolabs/applad/internal/databases"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/storage"
)

// appladSource reads a project on THIS Applad instance (the "duplicate a
// project" / same-instance import case). Users are read directly from the DB so
// their password credential (hash + algorithm + params) carries over intact;
// databases and storage are read through the service layer so schema and file
// access stay correct. A remote/cross-instance Applad source (endpoint + API
// key, for cloud <-> self-hosted) plugs into this same interface later.
type appladSource struct {
	projectID string // the SOURCE project on this instance
	db        *db.DB
	dbs       *databases.Service
	stg       *storage.Service
}

// NewAppladSource builds a same-instance Applad source for sourceProjectID.
func NewAppladSource(sourceProjectID string, database *db.DB, d *databases.Service, s *storage.Service) Source {
	return &appladSource{projectID: sourceProjectID, db: database, dbs: d, stg: s}
}

func (s *appladSource) Name() string { return "applad" }
func (s *appladSource) Close() error { return nil }

func (s *appladSource) Report(ctx context.Context, groups []Group) (map[Group]int, error) {
	out := map[Group]int{}
	for _, g := range groups {
		switch g {
		case GroupAuth:
			var n int
			if err := s.db.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM users WHERE project_id = $1", s.projectID).Scan(&n); err != nil {
				return nil, err
			}
			out[GroupAuth] = n
		case GroupDatabases:
			dbs, _, err := s.dbs.ListDatabases(ctx, s.projectID)
			if err != nil {
				return nil, err
			}
			total := len(dbs)
			for _, d := range dbs {
				tables, _, err := s.dbs.ListTables(ctx, d.ID, s.projectID)
				if err != nil {
					return nil, err
				}
				total += len(tables)
				for _, t := range tables {
					cols, _ := s.dbs.ListColumns(ctx, t.ID)
					idxs, _ := s.dbs.ListIndexes(ctx, t.ID)
					_, rowTotal, _ := s.dbs.ListRows(ctx, s.projectID, d.ID, t.ID, 1, 0)
					total += len(cols) + len(idxs) + rowTotal
				}
			}
			out[GroupDatabases] = total
		case GroupStorage:
			buckets, _, err := s.stg.ListBuckets(ctx, s.projectID)
			if err != nil {
				return nil, err
			}
			total := len(buckets)
			for _, b := range buckets {
				_, fileTotal, _ := s.stg.ListFiles(ctx, s.projectID, b.ID, 1, 0)
				total += fileTotal
			}
			out[GroupStorage] = total
		}
	}
	return out, nil
}

func (s *appladSource) Export(ctx context.Context, groups []Group, emit Emit) error {
	for _, g := range groups {
		var err error
		switch g {
		case GroupAuth:
			err = s.exportAuth(ctx, emit)
		case GroupDatabases:
			err = s.exportDatabases(ctx, emit)
		case GroupStorage:
			err = s.exportStorage(ctx, emit)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *appladSource) exportAuth(ctx context.Context, emit Emit) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, COALESCE(email,''), COALESCE(phone,''), COALESCE(name,''),
		        COALESCE(password_hash,''), COALESCE(password_algo,'bcrypt'), password_params,
		        email_verified, phone_verified, labels, prefs
		 FROM users WHERE project_id = $1`, s.projectID)
	if err != nil {
		return err
	}
	defer rows.Close()

	batch := make([]Resource, 0, 200)
	for rows.Next() {
		var u User
		var paramsRaw, labelsRaw, prefsRaw []byte
		if err := rows.Scan(&u.ID, &u.Email, &u.Phone, &u.Name,
			&u.PasswordHash, &u.PasswordAlgo, &paramsRaw,
			&u.EmailVerified, &u.PhoneVerified, &labelsRaw, &prefsRaw); err != nil {
			return err
		}
		if len(paramsRaw) > 0 {
			_ = json.Unmarshal(paramsRaw, &u.PasswordParams)
		}
		if len(labelsRaw) > 0 {
			_ = json.Unmarshal(labelsRaw, &u.Labels)
		}
		if len(prefsRaw) > 0 {
			_ = json.Unmarshal(prefsRaw, &u.Prefs)
		}
		batch = append(batch, u)
		if len(batch) >= 200 {
			if err := emit(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(batch) > 0 {
		return emit(ctx, batch)
	}
	return nil
}

func (s *appladSource) exportDatabases(ctx context.Context, emit Emit) error {
	dbs, _, err := s.dbs.ListDatabases(ctx, s.projectID)
	if err != nil {
		return err
	}
	for _, d := range dbs {
		if err := emit(ctx, []Resource{Database{ID: d.ID, Name: d.Name}}); err != nil {
			return err
		}
		tables, _, err := s.dbs.ListTables(ctx, d.ID, s.projectID)
		if err != nil {
			return err
		}
		for _, t := range tables {
			if err := emit(ctx, []Resource{Table{
				DatabaseID: d.ID, ID: t.ID, Name: t.Name,
				Permissions: t.Permissions, RowSecurity: t.RowSecurity,
			}}); err != nil {
				return err
			}
			cols, _ := s.dbs.ListColumns(ctx, t.ID)
			for _, c := range cols {
				if err := emit(ctx, []Resource{Column{
					DatabaseID: d.ID, TableID: t.ID, Key: c.Key, Type: c.Type,
					Required: c.Required, Array: c.Array, Default: c.Default, Options: c.Options,
				}}); err != nil {
					return err
				}
			}
			idxs, _ := s.dbs.ListIndexes(ctx, t.ID)
			for _, i := range idxs {
				if err := emit(ctx, []Resource{Index{
					DatabaseID: d.ID, TableID: t.ID, Key: i.Key, Type: i.Type,
					Columns: i.Columns, Orders: i.Orders,
				}}); err != nil {
					return err
				}
			}
			if err := s.exportRows(ctx, d.ID, t.ID, emit); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *appladSource) exportRows(ctx context.Context, databaseID, tableID string, emit Emit) error {
	const page = 500
	offset := 0
	for {
		rows, _, err := s.dbs.ListRows(ctx, s.projectID, databaseID, tableID, page, offset)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		batch := make([]Resource, 0, len(rows))
		for _, r := range rows {
			batch = append(batch, Row{
				DatabaseID: databaseID, TableID: tableID, ID: r.ID,
				Data: r.Data, Permissions: r.Permissions,
			})
		}
		if err := emit(ctx, batch); err != nil {
			return err
		}
		if len(rows) < page {
			return nil
		}
		offset += page
	}
}

func (s *appladSource) exportStorage(ctx context.Context, emit Emit) error {
	buckets, _, err := s.stg.ListBuckets(ctx, s.projectID)
	if err != nil {
		return err
	}
	for _, b := range buckets {
		if err := emit(ctx, []Resource{Bucket{
			ID: b.ID, Name: b.Name, Permissions: b.Permissions,
			FileSizeLimit: b.FileSizeLimit, AllowedMimeTypes: b.AllowedFileExtensions,
			FileSecurity: b.FileSecurity, Encryption: b.Encryption, Antivirus: b.Antivirus,
		}}); err != nil {
			return err
		}
		const page = 100
		offset := 0
		for {
			files, _, err := s.stg.ListFiles(ctx, s.projectID, b.ID, page, offset)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				break
			}
			for _, f := range files {
				// Bound memory: do not load a file larger than the import limit;
				// emit it with no content so the destination records a skip.
				if f.SizeOriginal > maxImportFileBytes {
					if err := emit(ctx, []Resource{File{BucketID: b.ID, ID: f.ID, Name: f.Name}}); err != nil {
						return err
					}
					continue
				}
				content, mime, err := s.stg.GetFileContent(ctx, f.ID, b.ID, s.projectID)
				if err != nil {
					// Skip an unreadable file rather than failing the whole job;
					// the destination records nothing and the count reflects it.
					continue
				}
				if mime == "" {
					mime = f.MimeType
				}
				if err := emit(ctx, []Resource{File{
					BucketID: b.ID, ID: f.ID, Name: f.Name, MimeType: mime,
					Content: content, Permissions: f.Permissions,
				}}); err != nil {
					return err
				}
			}
			if len(files) < page {
				break
			}
			offset += page
		}
	}
	return nil
}
