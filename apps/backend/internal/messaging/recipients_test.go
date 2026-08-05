package messaging

import (
	"context"
	"errors"
	"testing"
)

// The recipient caps reject an oversized send before any I/O, so one request
// cannot fan out to an unbounded (billable) list.
func TestRecipientCaps(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()
	many := make([]string, maxEmailRecipients+2)
	for i := range many {
		many[i] = "x"
	}

	if err := svc.SendSMSMulti(ctx, "p", many[:maxSMSRecipients+1], "hi"); !errors.Is(err, ErrTooManyRecipients) {
		t.Fatalf("SMS over cap should be rejected, got %v", err)
	}
	if err := svc.SendPushMulti(ctx, "p", many[:maxPushRecipients+1], "t", "b", nil); !errors.Is(err, ErrTooManyRecipients) {
		t.Fatalf("push over cap should be rejected, got %v", err)
	}
	if err := svc.SendEmail(ctx, many[:maxEmailRecipients+1], "s", "b"); !errors.Is(err, ErrTooManyRecipients) {
		t.Fatalf("email over cap should be rejected, got %v", err)
	}
}
