package plan

import (
	"strings"
	"testing"
)

// An update that cannot tell "not mentioned" from "cleared" erases whatever it
// was not told about, which is how a PATCH of one field wipes the rest.
func TestApplyInputChangesOnlyWhatWasNamed(t *testing.T) {
	item := &Item{Title: "Add promotions", Body: "the long version",
		Status: "in_progress", Priority: "high", Labels: []string{"checkout"}}

	status := "done"
	applyInput(item, Input{Status: &status})

	if item.Status != "done" {
		t.Errorf("status = %q", item.Status)
	}
	if item.Title != "Add promotions" || item.Body != "the long version" {
		t.Errorf("an unmentioned field was overwritten: %+v", item)
	}
	if item.Priority != "high" || len(item.Labels) != 1 {
		t.Errorf("an unmentioned field was overwritten: %+v", item)
	}

	// Naming a field with an empty value does clear it — that is the point of
	// the distinction.
	empty := ""
	applyInput(item, Input{Body: &empty})
	if item.Body != "" {
		t.Errorf("an explicit clear was ignored: %q", item.Body)
	}
}

func TestValidateRejectsUnknownStatesInTheCallersTerms(t *testing.T) {
	bad := "shipped"
	err := Input{Status: &bad}.Validate()
	if err == nil {
		t.Fatal("an unknown status was accepted")
	}
	// The message has to say what is allowed, or the caller has to guess.
	for _, s := range Statuses {
		if !contains(Statuses, s) {
			t.Fatalf("status list is inconsistent: %q", s)
		}
	}
	if got := err.Error(); !strings.Contains(got, "status must be one of") {
		t.Errorf("unhelpful message: %q", got)
	}

	blank := "   "
	if (Input{Title: &blank}).Validate() == nil {
		t.Error("a blank title was accepted")
	}
}

func TestClosedCoversBothWaysWorkEnds(t *testing.T) {
	// Cancelled work is finished too — a backlog that keeps showing it is not
	// a backlog.
	for _, s := range []string{"done", "cancelled"} {
		if !Closed(s) {
			t.Errorf("%q should count as closed", s)
		}
	}
	for _, s := range []string{"todo", "in_progress", "blocked"} {
		if Closed(s) {
			t.Errorf("%q should not count as closed", s)
		}
	}
}
