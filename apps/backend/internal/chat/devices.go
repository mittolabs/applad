package chat

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/uid"
)

// OneTimePrekeyInput is one entry in a batch of one-time prekeys a device
// uploads at registration time or when topping up its pool.
type OneTimePrekeyInput struct {
	KeyID     int    `json:"keyId"`
	PublicKey string `json:"publicKey"`
}

// RegisterDeviceInput carries a device's public key material at registration.
// Re-registering an existing device id (e.g. app reinstall reusing the same
// generated identity) updates its key material in place rather than erroring.
type RegisterDeviceInput struct {
	DeviceID        string
	Name            string
	RegistrationID  int
	IdentityKey     string
	SignedPrekeyID  int
	SignedPrekey    string
	SignedPrekeySig string
	PushToken       string
	PushProvider    string
	OneTimePrekeys  []OneTimePrekeyInput
}

// RegisterDevice publishes a device's public identity/prekey bundle. Private
// key material never reaches the server — only what's needed for another
// device to start an X3DH session with this one.
func (s *Service) RegisterDevice(ctx context.Context, projectID, userID string, in RegisterDeviceInput) (*model.Device, error) {
	if in.IdentityKey == "" || in.SignedPrekey == "" || in.SignedPrekeySig == "" {
		return nil, fmt.Errorf("chat: identityKey, signedPrekey, and signedPrekeySig are required")
	}
	id := uid.New(in.DeviceID)
	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("chat: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx,
		`INSERT INTO chat_devices (
			id, project_id, user_id, name, registration_id, identity_key,
			signed_prekey_id, signed_prekey, signed_prekey_sig, push_token, push_provider,
			status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'active', $12, $12)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			registration_id = EXCLUDED.registration_id,
			identity_key = EXCLUDED.identity_key,
			signed_prekey_id = EXCLUDED.signed_prekey_id,
			signed_prekey = EXCLUDED.signed_prekey,
			signed_prekey_sig = EXCLUDED.signed_prekey_sig,
			push_token = EXCLUDED.push_token,
			push_provider = EXCLUDED.push_provider,
			status = 'active',
			updated_at = EXCLUDED.updated_at`,
		id, projectID, userID, in.Name, in.RegistrationID, in.IdentityKey,
		in.SignedPrekeyID, in.SignedPrekey, in.SignedPrekeySig, nullableString(in.PushToken), nullableString(in.PushProvider),
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("chat: register device: %w", err)
	}

	for _, pk := range in.OneTimePrekeys {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chat_one_time_prekeys (id, device_id, project_id, key_id, public_key, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (device_id, key_id) DO NOTHING`,
			uid.New("unique()"), id, projectID, pk.KeyID, pk.PublicKey, now,
		); err != nil {
			return nil, fmt.Errorf("chat: insert one-time prekey: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("chat: commit: %w", err)
	}

	return &model.Device{
		ID: id, CreatedAt: now, UpdatedAt: now, UserID: userID, Name: in.Name,
		RegistrationID: in.RegistrationID, IdentityKey: in.IdentityKey,
		SignedPrekeyID: in.SignedPrekeyID, SignedPrekey: in.SignedPrekey, SignedPrekeySig: in.SignedPrekeySig,
		PushProvider: in.PushProvider, Status: "active",
	}, nil
}

// ListDevices returns the caller's own devices (for a "manage linked
// devices" screen), newest first.
func (s *Service) ListDevices(ctx context.Context, projectID, userID string) ([]*model.Device, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, created_at, updated_at, name, registration_id, identity_key,
		        signed_prekey_id, signed_prekey, signed_prekey_sig, COALESCE(push_provider, ''),
		        status, last_seen_at
		 FROM chat_devices
		 WHERE project_id = $1 AND user_id = $2
		 ORDER BY created_at DESC`,
		projectID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("chat: list devices: %w", err)
	}
	defer rows.Close()

	devices := make([]*model.Device, 0)
	for rows.Next() {
		var d model.Device
		var lastSeen sql.NullTime
		if err := rows.Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt, &d.Name, &d.RegistrationID, &d.IdentityKey,
			&d.SignedPrekeyID, &d.SignedPrekey, &d.SignedPrekeySig, &d.PushProvider, &d.Status, &lastSeen); err != nil {
			return nil, fmt.Errorf("chat: scan device: %w", err)
		}
		if lastSeen.Valid {
			t := lastSeen.Time
			d.LastSeenAt = &t
		}
		devices = append(devices, &d)
	}
	return devices, rows.Err()
}

// RevokeDevice unlinks one of the caller's own devices (status='revoked').
// History is kept — a revoked device's past messages remain valid — only new
// sessions and prekey-bundle fetches are refused going forward (enforced by
// GetPrekeyBundle/deviceOwnedByUser checking status='active').
func (s *Service) RevokeDevice(ctx context.Context, projectID, userID, deviceID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE chat_devices SET status = 'revoked', updated_at = NOW()
		 WHERE id = $1 AND project_id = $2 AND user_id = $3`,
		deviceID, projectID, userID,
	)
	if err != nil {
		return fmt.Errorf("chat: revoke device: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// TopUpPrekeys adds more one-time prekeys to a device's pool. Called by the
// client when GetPrekeyBundle reports the pool running low.
func (s *Service) TopUpPrekeys(ctx context.Context, projectID, userID, deviceID string, prekeys []OneTimePrekeyInput) error {
	owned, err := s.deviceOwnedByUser(ctx, projectID, deviceID, userID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrForbidden
	}
	now := time.Now().UTC()
	for _, pk := range prekeys {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO chat_one_time_prekeys (id, device_id, project_id, key_id, public_key, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (device_id, key_id) DO NOTHING`,
			uid.New("unique()"), deviceID, projectID, pk.KeyID, pk.PublicKey, now,
		); err != nil {
			return fmt.Errorf("chat: top up prekeys: %w", err)
		}
	}
	return nil
}

// GetPrekeyBundle returns deviceID's public key bundle for X3DH session
// establishment, atomically consuming one one-time prekey from its pool (if
// any remain — a bundle without one still lets X3DH proceed, just with one
// fewer Diffie-Hellman term, matching the Signal spec's optional OPK).
func (s *Service) GetPrekeyBundle(ctx context.Context, projectID, deviceID string) (*model.PrekeyBundle, error) {
	var b model.PrekeyBundle
	b.DeviceID = deviceID
	err := s.db.QueryRowContext(ctx,
		`SELECT registration_id, identity_key, signed_prekey_id, signed_prekey, signed_prekey_sig
		 FROM chat_devices WHERE id = $1 AND project_id = $2 AND status = 'active'`,
		deviceID, projectID,
	).Scan(&b.RegistrationID, &b.IdentityKey, &b.SignedPrekeyID, &b.SignedPrekey, &b.SignedPrekeySig)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("chat: get device: %w", err)
	}

	// Atomic consume: SKIP LOCKED so concurrent bundle fetches for the same
	// device (rare, but possible if several peers start a session at once)
	// each get a distinct one-time prekey rather than racing on the same row.
	var keyID int
	var publicKey string
	err = s.db.QueryRowContext(ctx,
		`DELETE FROM chat_one_time_prekeys
		 WHERE id = (
		     SELECT id FROM chat_one_time_prekeys
		     WHERE device_id = $1
		     ORDER BY created_at ASC
		     LIMIT 1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING key_id, public_key`,
		deviceID,
	).Scan(&keyID, &publicKey)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("chat: consume one-time prekey: %w", err)
	}
	if err == nil {
		b.OneTimePrekeyID = &keyID
		b.OneTimePrekey = publicKey
	}

	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chat_one_time_prekeys WHERE device_id = $1`,
		deviceID,
	).Scan(&b.PrekeysRemaining); err != nil {
		return nil, fmt.Errorf("chat: count remaining prekeys: %w", err)
	}

	return &b, nil
}

// ListUserDevices returns a target user's active device ids — used to
// discover which devices to encrypt to when starting or continuing a
// conversation with them. Deliberately minimal: no key material, since a
// session is established per-device via GetPrekeyBundle, which also performs
// the one-time-prekey consumption that listing must not.
func (s *Service) ListUserDevices(ctx context.Context, projectID, targetUserID string) ([]model.DeviceRef, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, registration_id, status FROM chat_devices
		 WHERE project_id = $1 AND user_id = $2 AND status = 'active'
		 ORDER BY created_at ASC`,
		projectID, targetUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("chat: list user devices: %w", err)
	}
	defer rows.Close()

	refs := make([]model.DeviceRef, 0)
	for rows.Next() {
		var d model.DeviceRef
		if err := rows.Scan(&d.ID, &d.RegistrationID, &d.Status); err != nil {
			return nil, fmt.Errorf("chat: scan device ref: %w", err)
		}
		refs = append(refs, d)
	}
	return refs, rows.Err()
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
