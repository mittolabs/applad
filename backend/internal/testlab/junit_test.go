package testlab

import (
	"strings"
	"testing"
)

// Vitest and Jest wrap suites in <testsuites> and put the meaningful grouping
// in classname.
func TestParsesVitestStyleReport(t *testing.T) {
	report := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="vitest tests" tests="3" failures="1" time="1.234">
  <testsuite name="src/cart.test.ts" tests="3" failures="1" time="1.234">
    <testcase classname="cart" name="adds an item" time="0.012"/>
    <testcase classname="cart" name="applies a discount" time="0.900">
      <failure message="expected 90 to be 100">AssertionError: expected 90 to be 100
    at cart.test.ts:14:5</failure>
    </testcase>
    <testcase classname="cart" name="handles empty carts" time="0.001">
      <skipped/>
    </testcase>
  </testsuite>
</testsuites>`

	cases, err := ParseJUnit(strings.NewReader(report))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("got %d cases, want 3", len(cases))
	}

	if cases[0].Status != CasePassed || cases[0].Name != "adds an item" {
		t.Errorf("case 0 = %+v", cases[0])
	}
	if cases[0].DurationMs != 12 {
		t.Errorf("duration = %dms, want 12", cases[0].DurationMs)
	}
	if cases[1].Status != CaseFailed {
		t.Errorf("case 1 status = %s, want failed", cases[1].Status)
	}
	if cases[1].FailureMessage != "expected 90 to be 100" {
		t.Errorf("failure message = %q", cases[1].FailureMessage)
	}
	if !strings.Contains(cases[1].FailureDetails, "cart.test.ts:14:5") {
		t.Errorf("failure details lost the stack: %q", cases[1].FailureDetails)
	}
	if cases[2].Status != CaseSkipped {
		t.Errorf("case 2 status = %s, want skipped", cases[2].Status)
	}
	if cases[0].SuiteName != "cart" {
		t.Errorf("suite name = %q, want cart (classname beats testsuite name)", cases[0].SuiteName)
	}
}

// gotestsum and pytest emit a bare <testsuite> with no wrapper.
func TestParsesBareTestsuiteReport(t *testing.T) {
	report := `<testsuite name="pkg/cart" tests="2" failures="1">
  <testcase classname="pkg/cart" name="TestAdd" time="0.004"/>
  <testcase classname="pkg/cart" name="TestDiscount" time="0.006">
    <failure message="mismatch">want 100, got 90</failure>
  </testcase>
</testsuite>`

	cases, err := ParseJUnit(strings.NewReader(report))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 {
		t.Fatalf("got %d cases, want 2", len(cases))
	}
	if cases[1].Status != CaseFailed || cases[1].FailureMessage != "mismatch" {
		t.Errorf("case 1 = %+v", cases[1])
	}
}

// An <error> is the framework failing to run a test, not the test disagreeing
// with the code. A red run reads very differently depending on which it was.
func TestErrorsAreDistinguishedFromFailures(t *testing.T) {
	report := `<testsuite name="s">
  <testcase classname="s" name="broken import"><error message="ModuleNotFoundError">no module named x</error></testcase>
  <testcase classname="s" name="wrong answer"><failure message="assert 1 == 2">nope</failure></testcase>
</testsuite>`

	cases, err := ParseJUnit(strings.NewReader(report))
	if err != nil {
		t.Fatal(err)
	}
	if cases[0].Status != CaseErrored {
		t.Errorf("case 0 status = %s, want errored", cases[0].Status)
	}
	if cases[1].Status != CaseFailed {
		t.Errorf("case 1 status = %s, want failed", cases[1].Status)
	}

	// Both still count against the run.
	s := Summarise(cases)
	if s.Failed != 2 || s.Passed != 0 {
		t.Errorf("summary = %+v, want 2 failed", s)
	}
}

// Nested suites are legal and some runners use them for describe blocks.
func TestFlattensNestedSuites(t *testing.T) {
	report := `<testsuites>
  <testsuite name="checkout">
    <testsuite name="discounts">
      <testcase name="ten percent off" time="0.1"/>
    </testsuite>
  </testsuite>
</testsuites>`

	cases, err := ParseJUnit(strings.NewReader(report))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 {
		t.Fatalf("got %d cases, want 1", len(cases))
	}
	if cases[0].SuiteName != "checkout › discounts" {
		t.Errorf("suite name = %q, want the nesting preserved", cases[0].SuiteName)
	}
}

// Runners disagree about where the failure text lives; some set only the
// attribute, some only the body.
func TestFailureMessageFallsBackToBody(t *testing.T) {
	report := `<testsuite name="s">
  <testcase classname="s" name="t"><failure>first line of the problem
second line</failure></testcase>
</testsuite>`

	cases, err := ParseJUnit(strings.NewReader(report))
	if err != nil {
		t.Fatal(err)
	}
	if cases[0].FailureMessage != "first line of the problem" {
		t.Errorf("message = %q, want the first body line", cases[0].FailureMessage)
	}
}

func TestRejectsNonJUnitInput(t *testing.T) {
	for name, input := range map[string]string{
		"empty":     "",
		"not xml":   "PASS ok 0.1s",
		"other xml": `<html><body>nope</body></html>`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseJUnit(strings.NewReader(input)); err == nil {
				t.Errorf("accepted %q as a JUnit report", input)
			}
		})
	}
}

// Time is optional, and absent time must not be read as a failure to parse.
func TestMissingTimeIsNotAnError(t *testing.T) {
	report := `<testsuite name="s"><testcase classname="s" name="t"/></testsuite>`
	cases, err := ParseJUnit(strings.NewReader(report))
	if err != nil {
		t.Fatal(err)
	}
	if cases[0].DurationMs != 0 || cases[0].Status != CasePassed {
		t.Errorf("case = %+v", cases[0])
	}
}
