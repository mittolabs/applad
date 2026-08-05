package transfer

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/mittolabs/applad/internal/auth"
)

// NewAppwriteSource reads an Appwrite project over its server REST API.
//
// Credentials: endpoint (e.g. https://cloud.appwrite.io/v1), projectId, apiKey.
func NewAppwriteSource(creds map[string]any) (Source, error) {
	endpoint := strings.TrimRight(credStr(creds, "endpoint"), "/")
	projectID := credStr(creds, "projectId")
	apiKey := credStr(creds, "apiKey")
	if endpoint == "" || projectID == "" || apiKey == "" {
		return nil, fmt.Errorf("appwrite: endpoint, projectId and apiKey are required")
	}
	if !strings.HasSuffix(endpoint, "/v1") {
		endpoint += "/v1"
	}
	return &appwriteSource{
		base: endpoint,
		http: newHTTPJSON(map[string]string{
			"X-Appwrite-Project":         projectID,
			"X-Appwrite-Key":             apiKey,
			"X-Appwrite-Response-Format": "1.6.0",
		}),
	}, nil
}

type appwriteSource struct {
	base string
	http *httpJSON
}

func (s *appwriteSource) Name() string { return "appwrite" }
func (s *appwriteSource) Close() error { return nil }

func pageQuery(limit, offset int) string {
	return fmt.Sprintf("?queries[]=%s&queries[]=%s",
		url.QueryEscape(fmt.Sprintf("limit(%d)", limit)),
		url.QueryEscape(fmt.Sprintf("offset(%d)", offset)))
}

func (s *appwriteSource) Report(ctx context.Context, groups []Group) (map[Group]int, error) {
	out := map[Group]int{}
	for _, g := range groups {
		switch g {
		case GroupAuth:
			var r struct {
				Total int `json:"total"`
			}
			if err := s.http.getInto(ctx, s.base+"/users"+pageQuery(1, 0), &r); err != nil {
				return nil, err
			}
			out[GroupAuth] = r.Total
		case GroupDatabases:
			// A cheap proxy: number of databases + collections. Row totals are not
			// summed up front to avoid listing every collection twice.
			var dbs struct {
				Total int `json:"total"`
			}
			if err := s.http.getInto(ctx, s.base+"/databases"+pageQuery(1, 0), &dbs); err == nil {
				out[GroupDatabases] = dbs.Total
			}
		case GroupStorage:
			var b struct {
				Total int `json:"total"`
			}
			if err := s.http.getInto(ctx, s.base+"/storage/buckets"+pageQuery(1, 0), &b); err == nil {
				out[GroupStorage] = b.Total
			}
		}
	}
	return out, nil
}

func (s *appwriteSource) Export(ctx context.Context, groups []Group, emit Emit) error {
	for _, g := range groups {
		var err error
		switch g {
		case GroupAuth:
			err = s.exportUsers(ctx, emit)
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

func (s *appwriteSource) exportUsers(ctx context.Context, emit Emit) error {
	offset := 0
	for {
		var r struct {
			Users []struct {
				ID                string         `json:"$id"`
				Email             string         `json:"email"`
				Phone             string         `json:"phone"`
				Name              string         `json:"name"`
				EmailVerification bool           `json:"emailVerification"`
				PhoneVerification bool           `json:"phoneVerification"`
				Password          string         `json:"password"`
				Hash              string         `json:"hash"`
				HashOptions       map[string]any `json:"hashOptions"`
				Prefs             map[string]any `json:"prefs"`
				Labels            []string       `json:"labels"`
			} `json:"users"`
		}
		if err := s.http.getInto(ctx, s.base+"/users"+pageQuery(100, offset), &r); err != nil {
			return err
		}
		if len(r.Users) == 0 {
			return nil
		}
		batch := make([]Resource, 0, len(r.Users))
		for _, u := range r.Users {
			algo, params := mapAppwriteHash(u.Hash, u.HashOptions)
			batch = append(batch, User{
				ID: u.ID, Email: u.Email, Phone: u.Phone, Name: u.Name,
				EmailVerified: u.EmailVerification, PhoneVerified: u.PhoneVerification,
				PasswordHash: u.Password, PasswordAlgo: algo, PasswordParams: params,
				Labels: u.Labels, Prefs: u.Prefs,
			})
		}
		if err := emit(ctx, batch); err != nil {
			return err
		}
		if len(r.Users) < 100 {
			return nil
		}
		offset += 100
	}
}

func (s *appwriteSource) exportDatabases(ctx context.Context, emit Emit) error {
	var dbs struct {
		Databases []struct {
			ID   string `json:"$id"`
			Name string `json:"name"`
		} `json:"databases"`
	}
	if err := s.http.getInto(ctx, s.base+"/databases"+pageQuery(100, 0), &dbs); err != nil {
		return err
	}
	for _, d := range dbs.Databases {
		if err := emit(ctx, []Resource{Database{ID: d.ID, Name: d.Name}}); err != nil {
			return err
		}
		var cols struct {
			Collections []struct {
				ID          string   `json:"$id"`
				Name        string   `json:"name"`
				Permissions []string `json:"$permissions"`
			} `json:"collections"`
		}
		if err := s.http.getInto(ctx, s.base+"/databases/"+d.ID+"/collections"+pageQuery(100, 0), &cols); err != nil {
			return err
		}
		for _, c := range cols.Collections {
			if err := emit(ctx, []Resource{Table{DatabaseID: d.ID, ID: c.ID, Name: c.Name, Permissions: c.Permissions}}); err != nil {
				return err
			}
			if err := s.exportAttributes(ctx, d.ID, c.ID, emit); err != nil {
				return err
			}
			if err := s.exportDocuments(ctx, d.ID, c.ID, emit); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *appwriteSource) exportAttributes(ctx context.Context, dbID, colID string, emit Emit) error {
	var r struct {
		Attributes []struct {
			Key      string `json:"key"`
			Type     string `json:"type"`
			Required bool   `json:"required"`
			Array    bool   `json:"array"`
			Default  any    `json:"default"`
		} `json:"attributes"`
	}
	if err := s.http.getInto(ctx, s.base+"/databases/"+dbID+"/collections/"+colID+"/attributes"+pageQuery(100, 0), &r); err != nil {
		return err
	}
	for _, a := range r.Attributes {
		if a.Type == "relationship" {
			continue // relationships are re-established structurally, not as columns
		}
		if err := emit(ctx, []Resource{Column{
			DatabaseID: dbID, TableID: colID, Key: a.Key, Type: mapColumnType(a.Type),
			Required: a.Required, Array: a.Array, Default: a.Default,
		}}); err != nil {
			return err
		}
	}
	return nil
}

func (s *appwriteSource) exportDocuments(ctx context.Context, dbID, colID string, emit Emit) error {
	offset := 0
	for {
		var r struct {
			Documents []map[string]any `json:"documents"`
		}
		if err := s.http.getInto(ctx, s.base+"/databases/"+dbID+"/collections/"+colID+"/documents"+pageQuery(100, offset), &r); err != nil {
			return err
		}
		if len(r.Documents) == 0 {
			return nil
		}
		batch := make([]Resource, 0, len(r.Documents))
		for _, doc := range r.Documents {
			id, _ := doc["$id"].(string)
			batch = append(batch, Row{DatabaseID: dbID, TableID: colID, ID: id, Data: stripSystemKeys(doc)})
		}
		if err := emit(ctx, batch); err != nil {
			return err
		}
		if len(r.Documents) < 100 {
			return nil
		}
		offset += 100
	}
}

func (s *appwriteSource) exportStorage(ctx context.Context, emit Emit) error {
	var bs struct {
		Buckets []struct {
			ID                    string   `json:"$id"`
			Name                  string   `json:"name"`
			Permissions           []string `json:"$permissions"`
			MaximumFileSize       int64    `json:"maximumFileSize"`
			AllowedFileExtensions []string `json:"allowedFileExtensions"`
			FileSecurity          bool     `json:"fileSecurity"`
			Encryption            bool     `json:"encryption"`
			Antivirus             bool     `json:"antivirus"`
		} `json:"buckets"`
	}
	if err := s.http.getInto(ctx, s.base+"/storage/buckets"+pageQuery(100, 0), &bs); err != nil {
		return err
	}
	for _, b := range bs.Buckets {
		if err := emit(ctx, []Resource{Bucket{
			ID: b.ID, Name: b.Name, Permissions: b.Permissions,
			FileSizeLimit: b.MaximumFileSize, AllowedMimeTypes: b.AllowedFileExtensions,
			FileSecurity: b.FileSecurity, Encryption: b.Encryption, Antivirus: b.Antivirus,
		}}); err != nil {
			return err
		}
		offset := 0
		for {
			var fs struct {
				Files []struct {
					ID           string `json:"$id"`
					Name         string `json:"name"`
					MimeType     string `json:"mimeType"`
					SizeOriginal int64  `json:"sizeOriginal"`
				} `json:"files"`
			}
			if err := s.http.getInto(ctx, s.base+"/storage/buckets/"+b.ID+"/files"+pageQuery(100, offset), &fs); err != nil {
				return err
			}
			if len(fs.Files) == 0 {
				break
			}
			for _, f := range fs.Files {
				if f.SizeOriginal > maxImportFileBytes {
					if err := emit(ctx, []Resource{File{BucketID: b.ID, ID: f.ID, Name: f.Name}}); err != nil {
						return err
					}
					continue
				}
				data, mime, err := s.http.getBytes(ctx, s.base+"/storage/buckets/"+b.ID+"/files/"+f.ID+"/download", maxImportFileBytes)
				if err != nil {
					continue
				}
				if mime == "" {
					mime = f.MimeType
				}
				if err := emit(ctx, []Resource{File{BucketID: b.ID, ID: f.ID, Name: f.Name, MimeType: mime, Content: data}}); err != nil {
					return err
				}
			}
			if len(fs.Files) < 100 {
				break
			}
			offset += 100
		}
	}
	return nil
}

// stripSystemKeys removes Appwrite's $-prefixed system fields from a document,
// leaving the user's own fields for the row Data.
func stripSystemKeys(doc map[string]any) map[string]any {
	out := make(map[string]any, len(doc))
	for k, v := range doc {
		if strings.HasPrefix(k, "$") {
			continue
		}
		out[k] = v
	}
	return out
}

// mapAppwriteHash maps Appwrite's hash algorithm name + options to Applad's
// password algorithm id and verify params. bcrypt and argon2 (Appwrite's
// defaults) map exactly; scrypt/scryptMod are best-effort from the exposed
// options; an unrecognized algorithm is passed through so the account imports
// but must reset its password before it can sign in.
func mapAppwriteHash(hash string, opts map[string]any) (string, map[string]any) {
	switch strings.ToLower(strings.TrimSpace(hash)) {
	case "bcrypt", "":
		return auth.AlgoBcrypt, nil
	case "argon2":
		// Appwrite stores the full PHC-encoded string in `password`.
		return auth.AlgoArgon2id, nil
	case "scryptmod":
		return auth.AlgoScryptFirebase, map[string]any{
			"salt":          optStr(opts, "salt"),
			"saltSeparator": optStr(opts, "saltSeparator"),
			"signerKey":     optStr(opts, "signerKey"),
			"memCost":       optNum(opts, "costCpu", 14),
			"rounds":        optNum(opts, "costMemory", 8),
		}
	case "scrypt":
		return auth.AlgoScrypt, map[string]any{
			"N":      optNum(opts, "costCpu", 16384),
			"r":      optNum(opts, "costMemory", 8),
			"p":      optNum(opts, "costParallel", 1),
			"keyLen": optNum(opts, "length", 64),
			"salt":   optStr(opts, "salt"),
		}
	case "sha":
		return auth.AlgoSHA256, map[string]any{"order": "password"}
	case "md5":
		return auth.AlgoMD5, map[string]any{"order": "password"}
	case "plaintext":
		return auth.AlgoPlaintext, nil
	default:
		// phpass and anything else we cannot verify: keep the hash under its name
		// so verifyForeignPassword returns an error (login fails, account intact).
		return strings.ToLower(strings.TrimSpace(hash)), nil
	}
}

func optStr(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func optNum(m map[string]any, k string, def int) int {
	if m == nil {
		return def
	}
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}
