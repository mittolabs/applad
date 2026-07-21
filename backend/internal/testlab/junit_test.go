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

// A runner told to retry reports every attempt. The last one is the outcome,
// and a test that failed and then passed is flaky — a fact worth recording
// rather than a red run worth arguing about.
func TestRetriesCollapseIntoOneFlakyResult(t *testing.T) {
	cases := []Case{
		{SuiteName: "cart", Name: "checkout", Status: CaseFailed, FailureMessage: "timeout", DurationMs: 900},
		{SuiteName: "cart", Name: "checkout", Status: CasePassed, DurationMs: 400},
		{SuiteName: "cart", Name: "total", Status: CasePassed, DurationMs: 10},
	}

	merged := MergeRetries(cases)
	if len(merged) != 2 {
		t.Fatalf("got %d results, want 2 — attempts at one test are one result", len(merged))
	}
	if merged[0].Status != CasePassed {
		t.Errorf("status = %s, want the final attempt to stand", merged[0].Status)
	}
	if !merged[0].Flaky {
		t.Error("a test that failed then passed must be marked flaky")
	}
	if merged[0].Retries != 1 {
		t.Errorf("retries = %d, want 1", merged[0].Retries)
	}
	if merged[0].DurationMs != 1300 {
		t.Errorf("duration = %d, want both attempts counted", merged[0].DurationMs)
	}
	if merged[1].Flaky {
		t.Error("a test that passed first time is not flaky")
	}

	// Flaky passes are still passes; the run is green with a caveat.
	s := Summarise(merged)
	if s.Passed != 2 || s.Failed != 0 || s.Flaky != 1 {
		t.Errorf("summary = %+v, want 2 passed with 1 flaky", s)
	}
}

// A test that fails every attempt is simply failing.
func TestRepeatedFailureIsNotFlaky(t *testing.T) {
	merged := MergeRetries([]Case{
		{SuiteName: "s", Name: "t", Status: CaseFailed, FailureMessage: "first"},
		{SuiteName: "s", Name: "t", Status: CaseFailed, FailureMessage: "again"},
	})
	if merged[0].Flaky {
		t.Error("consistently failing is not flaky")
	}
	if merged[0].FailureMessage != "again" {
		t.Errorf("message = %q, want the last attempt's", merged[0].FailureMessage)
	}
}

// Runners shorten long output directory names. Playwright elides the middle
// and inserts a hash, so an artifact belonging to a test with a long name
// never matched the whole slug — and the recording was orphaned from the run
// it documented.
func TestArtifactMatchesTruncatedDirectoryName(t *testing.T) {
	testSlug := "a-visitor-lands-on-the-home-page"
	truncated := "a-visitor-lands-on-the-hom-21f4f-itor-lands-on-the-home-page/video-webm"

	if !matchesCase(truncated, testSlug) {
		t.Error("a truncated directory name must still match its test")
	}
	if !matchesCase("cart-checkout/video-webm", "cart-checkout") {
		t.Error("an untruncated name must still match")
	}
	if matchesCase("some-other-test-entirely/video-webm", testSlug) {
		t.Error("an unrelated artifact must not be attached")
	}
	// Short names carry too little signal to match on an edge alone.
	if matchesCase("aaaaaaaaaaaaaaaaaaaaaaaa-bbbb/video-webm", "short") {
		t.Error("a short slug must require a full match")
	}
}
