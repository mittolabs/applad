package entitlements

import (
	"context"
	"database/sql"
	"log/slog"
)

// Dismissals records which notices a user has cleared from their own view.
//
// This is core's concern, not the provider's: whoever supplies a notice decides
// what to say, and core decides what this user still needs to see. Keeping the
// two apart means a provider never has to know about users, and dismissal keeps
// working no matter where notices come from.
type Dismissals struct {
	db *sql.DB
}

// NewDismissals creates the store.
func NewDismissals(db *sql.DB) *Dismissals { return &Dismissals{db: db} }

// Dismiss records that a user cleared a notice. Idempotent: clicking twice, or
// on two devices, is not an error.
func (d *Dismissals) Dismiss(ctx context.Context, userID, noticeID string) error {
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO notice_dismissals (user_id, notice_id) VALUES ($1, $2)
		 ON CONFLICT (user_id, notice_id) DO NOTHING`, userID, noticeID)
	return err
}

// For returns the notice ids this user has dismissed.
//
// A failure returns nothing rather than an error: the cost of forgetting a
// dismissal is a banner someone has already seen, which is a far better outcome
// than failing the request that renders their console.
func (d *Dismissals) For(ctx context.Context, userID string) map[string]bool {
	out := map[string]bool{}
	if userID == "" {
		return out
	}
	rows, err := d.db.QueryContext(ctx,
		`SELECT notice_id FROM notice_dismissals WHERE user_id = $1`, userID)
	if err != nil {
		slog.Warn("entitlements: cannot read dismissals", "error", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out[id] = true
		}
	}
	return out
}

// Filter removes notices this user has already dismissed.
func Filter(doc Document, dismissed map[string]bool) Document {
	if len(dismissed) == 0 {
		return doc
	}
	kept := make([]Notice, 0, len(doc.Notices))
	for _, n := range doc.Notices {
		if !dismissed[n.ID] {
			kept = append(kept, n)
		}
	}
	doc.Notices = kept
	return doc
}
