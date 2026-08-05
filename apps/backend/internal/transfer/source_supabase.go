package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// NewSupabaseSource reads a Supabase project: tables and auth.users over the
// Postgres connection (transaction pooler), and Storage over the REST API with
// the service_role key.
//
// Credentials:
//
//	host, port, user, password   Postgres connection (from the pooler string)
//	database                     optional, defaults to "postgres"
//	projectUrl                   e.g. https://<ref>.supabase.co (for Storage)
//	serviceKey                   the service_role API key (for Storage)
func NewSupabaseSource(creds map[string]any) (Source, error) {
	host := credStr(creds, "host")
	user := credStr(creds, "user")
	password := credStr(creds, "password")
	if host == "" || user == "" || password == "" {
		return nil, fmt.Errorf("supabase: host, user and password are required")
	}
	port := credStr(creds, "port")
	if port == "" {
		port = "5432"
	}
	database := credStr(creds, "database")
	if database == "" {
		database = "postgres"
	}
	dsn, err := buildPGDSN(host, port, user, password, database)
	if err != nil {
		return nil, fmt.Errorf("supabase: %w", err)
	}
	db, err := openGuardedPostgres(dsn)
	if err != nil {
		return nil, err
	}

	src := &pgSource{
		name:       "supabase",
		db:         db,
		dataSchema: "public",
		// email_confirmed_at != NULL means the email is verified; name is pulled
		// from the user metadata Supabase stores.
		authSQL: `SELECT id::text,
		                 COALESCE(email,''),
		                 COALESCE(phone,''),
		                 COALESCE(raw_user_meta_data->>'full_name', raw_user_meta_data->>'name', ''),
		                 COALESCE(encrypted_password,''),
		                 (email_confirmed_at IS NOT NULL)
		          FROM auth.users`,
	}

	projectURL := strings.TrimRight(credStr(creds, "projectUrl"), "/")
	serviceKey := credStr(creds, "serviceKey")
	if projectURL != "" && serviceKey != "" {
		st := &supabaseStorage{
			base: projectURL,
			http: newHTTPJSON(map[string]string{
				"Authorization": "Bearer " + serviceKey,
				"apikey":        serviceKey,
			}),
		}
		src.storage = st.export
	}
	return src, nil
}

// supabaseStorage lists buckets and objects through the Storage REST API.
type supabaseStorage struct {
	base string
	http *httpJSON
}

func (s *supabaseStorage) export(ctx context.Context, emit Emit) (int, error) {
	var buckets []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := s.http.getInto(ctx, s.base+"/storage/v1/bucket", &buckets); err != nil {
		return 0, fmt.Errorf("supabase: list buckets: %w", err)
	}
	count := 0
	for _, b := range buckets {
		id := b.ID
		if id == "" {
			id = b.Name
		}
		if err := emit(ctx, []Resource{Bucket{ID: id, Name: b.Name}}); err != nil {
			return count, err
		}
		offset := 0
		for {
			body, _ := json.Marshal(map[string]any{"prefix": "", "limit": 100, "offset": offset})
			var objs []struct {
				Name     string `json:"name"`
				Metadata struct {
					Size     int64  `json:"size"`
					MimeType string `json:"mimetype"`
				} `json:"metadata"`
			}
			if err := s.http.do(ctx, "POST", s.base+"/storage/v1/object/list/"+id, bytes.NewReader(body), &objs); err != nil {
				return count, fmt.Errorf("supabase: list objects: %w", err)
			}
			if len(objs) == 0 {
				break
			}
			for _, o := range objs {
				if o.Name == "" || strings.HasSuffix(o.Name, "/") {
					continue // folder placeholder
				}
				if o.Metadata.Size > maxImportFileBytes {
					if err := emit(ctx, []Resource{File{BucketID: id, ID: o.Name, Name: o.Name}}); err != nil {
						return count, err
					}
					continue
				}
				data, mime, err := s.http.getBytes(ctx, s.base+"/storage/v1/object/"+id+"/"+url.PathEscape(o.Name), maxImportFileBytes)
				if err != nil {
					continue // skip an unreadable object rather than failing the job
				}
				if mime == "" {
					mime = o.Metadata.MimeType
				}
				if err := emit(ctx, []Resource{File{BucketID: id, ID: o.Name, Name: o.Name, MimeType: mime, Content: data}}); err != nil {
					return count, err
				}
				count++
			}
			if len(objs) < 100 {
				break
			}
			offset += 100
		}
	}
	return count, nil
}

func credStr(creds map[string]any, key string) string {
	if creds == nil {
		return ""
	}
	if v, ok := creds[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
