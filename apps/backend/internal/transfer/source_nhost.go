package transfer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// NewNhostSource reads an NHost (hasura-auth) project over its Postgres
// connection: public-schema tables and the auth.users accounts (bcrypt).
//
// Credentials: host, port, user, password, database (defaults to the user).
// Storage is imported when storageUrl + adminSecret are supplied: file metadata
// is read from the storage.* tables over the same Postgres connection, and the
// bytes are downloaded from the hasura-storage API.
func NewNhostSource(creds map[string]any) (Source, error) {
	host := credStr(creds, "host")
	user := credStr(creds, "user")
	password := credStr(creds, "password")
	if host == "" || user == "" || password == "" {
		return nil, fmt.Errorf("nhost: host, user and password are required")
	}
	port := credStr(creds, "port")
	if port == "" {
		port = "5432"
	}
	database := credStr(creds, "database")
	if database == "" {
		database = user
	}
	dsn, err := buildPGDSN(host, port, user, password, database)
	if err != nil {
		return nil, fmt.Errorf("nhost: %w", err)
	}
	db, err := openGuardedPostgres(dsn)
	if err != nil {
		return nil, err
	}
	src := &pgSource{
		name:       "nhost",
		db:         db,
		dataSchema: "public",
		authSQL: `SELECT id::text,
		                 COALESCE(email,''),
		                 COALESCE(phone_number,''),
		                 COALESCE(display_name,''),
		                 COALESCE(password_hash,''),
		                 COALESCE(email_verified,false)
		          FROM auth.users`,
	}

	storageURL := strings.TrimRight(credStr(creds, "storageUrl"), "/")
	adminSecret := credStr(creds, "adminSecret")
	if storageURL != "" && adminSecret != "" {
		st := &nhostStorage{
			db:   db,
			base: storageURL,
			http: newHTTPJSON(map[string]string{"x-hasura-admin-secret": adminSecret}),
		}
		src.storage = st.export
	}
	return src, nil
}

// nhostStorage reads file metadata from the storage.* tables (over the same
// Postgres connection the source already holds) and downloads bytes from the
// hasura-storage service.
type nhostStorage struct {
	db   *sql.DB
	base string
	http *httpJSON
}

func (s *nhostStorage) export(ctx context.Context, emit Emit) (int, error) {
	buckets, err := s.listBuckets(ctx)
	if err != nil {
		// A project may have hasura-auth but no storage schema; treat as empty.
		return 0, nil
	}
	count := 0
	for _, b := range buckets {
		if err := emit(ctx, []Resource{Bucket{ID: b, Name: b}}); err != nil {
			return count, err
		}
		rows, err := s.db.QueryContext(ctx,
			`SELECT id::text, COALESCE(name,''), COALESCE(size,0), COALESCE(mime_type,'')
			 FROM storage.files WHERE bucket_id = $1`, b)
		if err != nil {
			return count, fmt.Errorf("nhost: list files: %w", err)
		}
		var files []nhostFile
		for rows.Next() {
			var f nhostFile
			if err := rows.Scan(&f.id, &f.name, &f.size, &f.mime); err != nil {
				rows.Close()
				return count, err
			}
			files = append(files, f)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return count, err
		}
		for _, f := range files {
			name := f.name
			if name == "" {
				name = f.id
			}
			if f.size > maxImportFileBytes {
				if err := emit(ctx, []Resource{File{BucketID: b, ID: f.id, Name: name}}); err != nil {
					return count, err
				}
				continue
			}
			data, mime, derr := s.http.getBytes(ctx, s.base+"/v1/files/"+f.id, maxImportFileBytes)
			if derr != nil {
				continue // skip an unreadable object rather than failing the job
			}
			if mime == "" {
				mime = f.mime
			}
			if err := emit(ctx, []Resource{File{BucketID: b, ID: f.id, Name: name, MimeType: mime, Content: data}}); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

func (s *nhostStorage) listBuckets(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id::text FROM storage.buckets ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

type nhostFile struct {
	id   string
	name string
	size int64
	mime string
}
