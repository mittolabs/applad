package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/auth"
)

// NewFirebaseSource reads a Firebase project: Authentication (users with the
// modified-scrypt password hash), Firestore (collections -> tables, documents
// -> rows), and Cloud Storage (best-effort).
//
// Credentials:
//
//	serviceAccount   the service-account JSON (object or string) with
//	                 project_id, client_email, private_key
//	signerKey, saltSeparator, rounds, memCost
//	                 the project's password hash config (Firebase console ->
//	                 Authentication -> Users -> Password hash parameters).
//	                 Required to carry passwords; without them users import
//	                 without a usable password.
//	storageBucket    optional, defaults to <projectId>.appspot.com
func NewFirebaseSource(creds map[string]any) (Source, error) {
	sa, err := parseServiceAccount(creds["serviceAccount"])
	if err != nil {
		return nil, err
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(sa.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("firebase: parse service-account private key: %w", err)
	}
	bucket := credStr(creds, "storageBucket")
	if bucket == "" {
		bucket = sa.ProjectID + ".appspot.com"
	}
	return &firebaseSource{
		projectID:  sa.ProjectID,
		clientMail: sa.ClientEmail,
		key:        key,
		http:       newHTTPJSON(nil),
		bucket:     bucket,
		hashParams: map[string]any{
			"signerKey":     credStr(creds, "signerKey"),
			"saltSeparator": credStr(creds, "saltSeparator"),
			"rounds":        credNum(creds, "rounds", 8),
			"memCost":       credNum(creds, "memCost", 14),
		},
		haveHashConfig: credStr(creds, "signerKey") != "",
	}, nil
}

type serviceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

func parseServiceAccount(v any) (*serviceAccount, error) {
	var raw []byte
	switch t := v.(type) {
	case string:
		raw = []byte(t)
	case map[string]any:
		raw, _ = json.Marshal(t)
	default:
		return nil, fmt.Errorf("firebase: serviceAccount credential is required (JSON)")
	}
	var sa serviceAccount
	if err := json.Unmarshal(raw, &sa); err != nil {
		return nil, fmt.Errorf("firebase: invalid service-account JSON: %w", err)
	}
	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, fmt.Errorf("firebase: service account missing project_id/client_email/private_key")
	}
	return &sa, nil
}

type firebaseSource struct {
	projectID      string
	clientMail     string
	key            any
	http           *httpJSON
	bucket         string
	hashParams     map[string]any
	haveHashConfig bool

	token    string
	tokenExp time.Time
}

func (s *firebaseSource) Name() string { return "firebase" }
func (s *firebaseSource) Close() error { return nil }

// accessToken mints (and caches) a Google OAuth2 access token from the service
// account via a signed JWT assertion.
func (s *firebaseSource) accessToken(ctx context.Context) (string, error) {
	if s.token != "" && time.Now().Before(s.tokenExp.Add(-1*time.Minute)) {
		return s.token, nil
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   s.clientMail,
		"scope": "https://www.googleapis.com/auth/cloud-platform",
		"aud":   "https://oauth2.googleapis.com/token",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	assertion, err := tok.SignedString(s.key)
	if err != nil {
		return "", fmt.Errorf("firebase: sign assertion: %w", err)
	}
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req := newHTTPJSON(map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := req.do(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()), &out); err != nil {
		return "", fmt.Errorf("firebase: token exchange: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("firebase: empty access token")
	}
	s.token = out.AccessToken
	s.tokenExp = now.Add(time.Duration(out.ExpiresIn) * time.Second)
	s.http = newHTTPJSON(map[string]string{"Authorization": "Bearer " + s.token})
	return s.token, nil
}

func (s *firebaseSource) Report(ctx context.Context, groups []Group) (map[Group]int, error) {
	// Validate credentials by minting a token; counts fill in as export streams.
	if _, err := s.accessToken(ctx); err != nil {
		return nil, err
	}
	out := map[Group]int{}
	for _, g := range groups {
		out[g] = 0
	}
	return out, nil
}

func (s *firebaseSource) Export(ctx context.Context, groups []Group, emit Emit) error {
	if _, err := s.accessToken(ctx); err != nil {
		return err
	}
	for _, g := range groups {
		var err error
		switch g {
		case GroupAuth:
			err = s.exportUsers(ctx, emit)
		case GroupDatabases:
			err = s.exportFirestore(ctx, emit)
		case GroupStorage:
			err = s.exportStorage(ctx, emit)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *firebaseSource) exportUsers(ctx context.Context, emit Emit) error {
	pageToken := ""
	for {
		u := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/projects/%s/accounts:batchGet?maxResults=200", s.projectID)
		if pageToken != "" {
			u += "&nextPageToken=" + url.QueryEscape(pageToken)
		}
		var r struct {
			Users []struct {
				LocalID       string `json:"localId"`
				Email         string `json:"email"`
				PhoneNumber   string `json:"phoneNumber"`
				DisplayName   string `json:"displayName"`
				PasswordHash  string `json:"passwordHash"`
				Salt          string `json:"salt"`
				EmailVerified bool   `json:"emailVerified"`
			} `json:"users"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := s.http.getInto(ctx, u, &r); err != nil {
			return err
		}
		batch := make([]Resource, 0, len(r.Users))
		for _, fu := range r.Users {
			user := User{
				ID: fu.LocalID, Email: fu.Email, Phone: fu.PhoneNumber, Name: fu.DisplayName,
				EmailVerified: fu.EmailVerified,
			}
			// Firebase scrypt needs the per-user salt plus the project hash config.
			if fu.PasswordHash != "" && s.haveHashConfig {
				params := map[string]any{"salt": fu.Salt}
				for k, v := range s.hashParams {
					params[k] = v
				}
				user.PasswordHash = fu.PasswordHash
				user.PasswordAlgo = auth.AlgoScryptFirebase
				user.PasswordParams = params
			}
			batch = append(batch, user)
		}
		if len(batch) > 0 {
			if err := emit(ctx, batch); err != nil {
				return err
			}
		}
		// Stop on an empty page, no next token, or a non-advancing token (a
		// hostile server returning a constant token would loop forever).
		if r.NextPageToken == "" || r.NextPageToken == pageToken || len(r.Users) == 0 {
			return nil
		}
		pageToken = r.NextPageToken
	}
}

func (s *firebaseSource) exportFirestore(ctx context.Context, emit Emit) error {
	const dbID = "firestore"
	if err := emit(ctx, []Resource{Database{ID: dbID, Name: "firestore"}}); err != nil {
		return err
	}
	// List top-level collections.
	base := fmt.Sprintf("https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents", s.projectID)
	var ids struct {
		CollectionIds []string `json:"collectionIds"`
	}
	if err := s.http.do(ctx, "POST", base+":listCollectionIds", strings.NewReader("{}"), &ids); err != nil {
		return err
	}
	for _, coll := range ids.CollectionIds {
		if err := emit(ctx, []Resource{Table{DatabaseID: dbID, ID: coll, Name: coll}}); err != nil {
			return err
		}
		// Firestore is schemaless; columns are inferred from the first documents.
		seenCols := map[string]bool{}
		pageToken := ""
		for {
			u := base + "/" + url.PathEscape(coll) + "?pageSize=200"
			if pageToken != "" {
				u += "&pageToken=" + url.QueryEscape(pageToken)
			}
			var r struct {
				Documents []struct {
					Name   string                     `json:"name"`
					Fields map[string]json.RawMessage `json:"fields"`
				} `json:"documents"`
				NextPageToken string `json:"nextPageToken"`
			}
			if err := s.http.getInto(ctx, u, &r); err != nil {
				return err
			}
			if len(r.Documents) == 0 {
				break
			}
			rowBatch := make([]Resource, 0, len(r.Documents))
			for _, doc := range r.Documents {
				data := map[string]any{}
				for k, raw := range doc.Fields {
					data[k] = decodeFirestoreValue(raw)
					if !seenCols[k] {
						seenCols[k] = true
						if err := emit(ctx, []Resource{Column{
							DatabaseID: dbID, TableID: coll, Key: k, Type: "string",
						}}); err != nil {
							return err
						}
					}
				}
				id := doc.Name
				if i := strings.LastIndex(id, "/"); i >= 0 {
					id = id[i+1:]
				}
				rowBatch = append(rowBatch, Row{DatabaseID: dbID, TableID: coll, ID: id, Data: data})
			}
			if err := emit(ctx, rowBatch); err != nil {
				return err
			}
			if r.NextPageToken == "" || r.NextPageToken == pageToken {
				break
			}
			pageToken = r.NextPageToken
		}
	}
	return nil
}

// decodeFirestoreValue turns a Firestore typed value into a plain Go value.
func decodeFirestoreValue(raw json.RawMessage) any {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	for k, v := range m {
		switch k {
		case "stringValue", "timestampValue", "referenceValue", "bytesValue":
			var s string
			_ = json.Unmarshal(v, &s)
			return s
		case "integerValue":
			var s string
			_ = json.Unmarshal(v, &s)
			return s
		case "doubleValue":
			var f float64
			_ = json.Unmarshal(v, &f)
			return f
		case "booleanValue":
			var b bool
			_ = json.Unmarshal(v, &b)
			return b
		case "nullValue":
			return nil
		case "mapValue":
			var mv struct {
				Fields map[string]json.RawMessage `json:"fields"`
			}
			_ = json.Unmarshal(v, &mv)
			out := map[string]any{}
			for fk, fv := range mv.Fields {
				out[fk] = decodeFirestoreValue(fv)
			}
			return out
		case "arrayValue":
			var av struct {
				Values []json.RawMessage `json:"values"`
			}
			_ = json.Unmarshal(v, &av)
			out := make([]any, 0, len(av.Values))
			for _, e := range av.Values {
				out = append(out, decodeFirestoreValue(e))
			}
			return out
		}
	}
	return nil
}

func (s *firebaseSource) exportStorage(ctx context.Context, emit Emit) error {
	if err := emit(ctx, []Resource{Bucket{ID: "default", Name: s.bucket}}); err != nil {
		return err
	}
	pageToken := ""
	for {
		u := "https://storage.googleapis.com/storage/v1/b/" + url.PathEscape(s.bucket) + "/o?maxResults=200"
		if pageToken != "" {
			u += "&pageToken=" + url.QueryEscape(pageToken)
		}
		var r struct {
			Items []struct {
				Name        string `json:"name"`
				ContentType string `json:"contentType"`
				Size        string `json:"size"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := s.http.getInto(ctx, u, &r); err != nil {
			return err
		}
		for _, it := range r.Items {
			// Skip oversized objects before downloading (size is a decimal string).
			if n, err := strconv.ParseInt(it.Size, 10, 64); err == nil && n > maxImportFileBytes {
				if err := emit(ctx, []Resource{File{BucketID: "default", ID: it.Name, Name: it.Name}}); err != nil {
					return err
				}
				continue
			}
			dl := "https://storage.googleapis.com/storage/v1/b/" + url.PathEscape(s.bucket) + "/o/" + url.PathEscape(it.Name) + "?alt=media"
			data, mime, err := s.http.getBytes(ctx, dl, maxImportFileBytes)
			if err != nil {
				continue
			}
			if mime == "" {
				mime = it.ContentType
			}
			if err := emit(ctx, []Resource{File{BucketID: "default", ID: it.Name, Name: it.Name, MimeType: mime, Content: data}}); err != nil {
				return err
			}
		}
		if r.NextPageToken == "" || r.NextPageToken == pageToken {
			return nil
		}
		pageToken = r.NextPageToken
	}
}

func credNum(creds map[string]any, key string, def int) int {
	if creds == nil {
		return def
	}
	switch v := creds[key].(type) {
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
