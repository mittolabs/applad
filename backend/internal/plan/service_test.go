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

// The point of a grid rather than a dropdown is that the two axes are not
// symmetric: impact is a property of the problem and urgency a property of the
// calendar, so a soft deadline may defer high-impact work but must not demote
// it.
func TestDefaultGridLetsImpactDominateUrgency(t *testing.T) {
	rank := map[string]int{"low": 0, "medium": 1, "high": 2, "urgent": 3}

	for urgency := LevelLow; urgency <= LevelHigh; urgency++ {
		if got := DefaultGrid[[2]int{LevelHigh, urgency}]; rank[got] < rank["high"] {
			t.Errorf("high impact at urgency %d resolved to %q — impact was demoted", urgency, got)
		}
	}

	// Low survives only when nothing is at stake and nobody is waiting.
	if got := DefaultGrid[[2]int{LevelLow, LevelLow}]; got != "low" {
		t.Errorf("low/low = %q, want low", got)
	}
	for _, cell := range [][2]int{{1, 2}, {1, 3}, {2, 1}} {
		if got := DefaultGrid[cell]; got == "low" {
			t.Errorf("%v resolved to low — something was at stake or somebody was waiting", cell)
		}
	}

	// Only both answers at their highest earns the top of the scale.
	for cell, priority := range DefaultGrid {
		if priority == "urgent" && cell != [2]int{LevelHigh, LevelHigh} {
			t.Errorf("%v resolved to urgent without both answers high", cell)
		}
	}

	if len(DefaultGrid) != 9 {
		t.Errorf("the grid has %d cells, want 9", len(DefaultGrid))
	}
}
