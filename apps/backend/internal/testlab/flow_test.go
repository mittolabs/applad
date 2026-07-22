package testlab

import (
	"strings"
	"testing"
)

// The point of storing steps as data is that the same recording serves every
// platform. If a step compiles for the web but has no meaning on a device,
// the model has drifted.
func TestOneFlowCompilesForBothPlatforms(t *testing.T) {
	flow := Flow{
		Name:   "a visitor reaches the about page",
		Target: "http://the-range.applad.dev",
		Steps: []Step{
			{Kind: StepGoto, Value: "/", Description: "open the home page"},
			{
				Kind:        StepTap,
				Target:      Target{Role: "link", Name: "About Us", CSS: "nav > a:nth-child(3)"},
				Description: "tap About Us",
			},
			{Kind: StepExpectURL, Value: "/about.html", Description: "the about page opens"},
		},
	}

	web := CompilePlaywright(flow)
	if !strings.Contains(web, "getByRole('link', { name: 'About Us' })") {
		t.Errorf("web compile lost the role selector:\n%s", web)
	}
	if !strings.Contains(web, "toHaveURL('/about.html')") {
		t.Errorf("web compile lost the assertion:\n%s", web)
	}
	if !strings.Contains(web, "// tap About Us") {
		t.Errorf("web compile dropped the step's meaning:\n%s", web)
	}

	device := CompileMaestro(flow, "com.example.app")
	if !strings.Contains(device, `tapOn: "About Us"`) {
		t.Errorf("device compile lost the tap:\n%s", device)
	}
}

// A recorded click must not become a coordinate. The whole value of the studio
// is that it produces a selector somebody would have written by hand.
func TestLocatorPrefersTheMostDurableStrategy(t *testing.T) {
	tests := []struct {
		name   string
		target Target
		want   string
	}{
		{
			name:   "a test id beats everything",
			target: Target{TestID: "submit", Role: "button", Name: "Send", CSS: "#a > b"},
			want:   "getByTestId('submit')",
		},
		{
			name:   "role and name beat css",
			target: Target{Role: "button", Name: "Send", CSS: "#a > b"},
			want:   "getByRole('button', { name: 'Send' })",
		},
		{
			name:   "a label beats raw text",
			target: Target{Label: "Email", Text: "Email", CSS: "input"},
			want:   "getByLabel('Email')",
		},
		{
			name:   "css only when nothing better exists",
			target: Target{CSS: "div.hero > span"},
			want:   "page.locator('div.hero > span')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.locator(); !strings.Contains(got, tt.want) {
				t.Errorf("locator() = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

// Playwright fails rather than guessing when a locator matches several
// elements — exactly what broke the hand-written Range test. A recording knows
// which one was clicked, so it must say so.
func TestAmbiguousTargetsCarryTheirIndex(t *testing.T) {
	first := Target{Text: "+254", Nth: 0}
	if got := first.locator(); !strings.HasSuffix(got, ".first()") {
		t.Errorf("locator() = %q, want a .first() to survive strict mode", got)
	}

	third := Target{Text: "+254", Nth: 2}
	if got := third.locator(); !strings.HasSuffix(got, ".nth(2)") {
		t.Errorf("locator() = %q, want the recorded index", got)
	}
}

func TestQuotingSurvivesAwkwardText(t *testing.T) {
	flow := Flow{
		Name: "it's a test",
		Steps: []Step{
			{Kind: StepTap, Target: Target{Text: "Don't click"}, Description: "tap"},
		},
	}
	out := CompilePlaywright(flow)
	if strings.Contains(out, "'Don't click'") {
		t.Errorf("apostrophe was not escaped, output will not parse:\n%s", out)
	}
	if !strings.Contains(out, `Don\'t click`) {
		t.Errorf("expected an escaped apostrophe:\n%s", out)
	}
}
