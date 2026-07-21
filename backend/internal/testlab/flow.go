package testlab

import (
	"fmt"
	"strings"
)

/*
 * A recorded flow, and what it compiles to.
 *
 * Steps are stored as data rather than as code so that one recording studio
 * can serve every platform: the same list becomes a Playwright spec for the
 * web and a Maestro flow on a device. Adding a platform is a compiler, not a
 * second product.
 *
 * A step also records intent — "expect the contact number to be visible" —
 * rather than only mechanics, which is what will later let a specification
 * example be attached to the flow that verifies it.
 */

// StepKind is what a step does.
type StepKind string

const (
	StepGoto          StepKind = "goto"
	StepTap           StepKind = "tap"
	StepType          StepKind = "type"
	StepPress         StepKind = "press"
	StepExpectVisible StepKind = "expectVisible"
	StepExpectText    StepKind = "expectText"
	StepExpectURL     StepKind = "expectURL"
)

// Target is how a step finds its element.
//
// Several strategies are recorded at once because they age differently: a test
// id survives a redesign, a role and name survive a restructure, and a CSS
// path survives neither but always resolves. The compiler picks the most
// durable one available, and the rest remain as fallbacks for a future
// self-healing pass.
type Target struct {
	TestID string `json:"testId,omitempty"`
	Role   string `json:"role,omitempty"`
	Name   string `json:"name,omitempty"`
	Label  string `json:"label,omitempty"`
	Text   string `json:"text,omitempty"`

	Placeholder string `json:"placeholder,omitempty"`
	CSS         string `json:"css,omitempty"`
	// Nth disambiguates when a selector legitimately matches several elements,
	// which is common on a page with repeated cards or a carousel.
	Nth int `json:"nth,omitempty"`
}

// Step is one recorded action or assertion.
type Step struct {
	Kind   StepKind `json:"kind"`
	Target Target   `json:"target,omitempty"`
	Value  string   `json:"value,omitempty"`
	// Description is what the step means, in the words shown in the studio.
	Description string `json:"description"`
}

// Flow is a recording.
type Flow struct {
	ID        string `json:"$id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Platform  string `json:"platform"`
	Target    string `json:"target"`
	Steps     []Step `json:"steps"`
	RunnerID  string `json:"runnerId,omitempty"`
	TestID    string `json:"testId,omitempty"`
}

// locator renders a target as a Playwright locator, preferring the strategy
// that survives the most change.
func (t Target) locator() string {
	var base string
	switch {
	case t.TestID != "":
		base = fmt.Sprintf("page.getByTestId(%s)", quote(t.TestID))
	case t.Role != "" && t.Name != "":
		base = fmt.Sprintf("page.getByRole(%s, { name: %s })", quote(t.Role), quote(t.Name))
	case t.Label != "":
		base = fmt.Sprintf("page.getByLabel(%s)", quote(t.Label))
	case t.Placeholder != "":
		base = fmt.Sprintf("page.getByPlaceholder(%s)", quote(t.Placeholder))
	case t.Text != "":
		base = fmt.Sprintf("page.getByText(%s)", quote(t.Text))
	case t.Role != "":
		base = fmt.Sprintf("page.getByRole(%s)", quote(t.Role))
	default:
		base = fmt.Sprintf("page.locator(%s)", quote(t.CSS))
	}
	// Playwright is strict: a locator matching several elements fails rather
	// than guessing, so an index recorded at capture time is preserved.
	if t.Nth > 0 {
		return fmt.Sprintf("%s.nth(%d)", base, t.Nth)
	}
	if t.Nth == 0 && (t.Text != "" || t.Role != "") {
		return base + ".first()"
	}
	return base
}

// CompilePlaywright renders a flow as a Playwright spec.
func CompilePlaywright(f Flow) string {
	var b strings.Builder
	b.WriteString("// Recorded in Applad. Edit freely: this is ordinary Playwright.\n")
	b.WriteString("const { test, expect } = require('@playwright/test');\n\n")
	b.WriteString(fmt.Sprintf("test(%s, async ({ page }) => {\n", quote(f.Name)))

	for _, s := range f.Steps {
		if s.Description != "" {
			b.WriteString(fmt.Sprintf("  // %s\n", s.Description))
		}
		switch s.Kind {
		case StepGoto:
			b.WriteString(fmt.Sprintf("  await page.goto(%s);\n", quote(s.Value)))
		case StepTap:
			b.WriteString(fmt.Sprintf("  await %s.click();\n", s.Target.locator()))
		case StepType:
			b.WriteString(fmt.Sprintf("  await %s.fill(%s);\n", s.Target.locator(), quote(s.Value)))
		case StepPress:
			b.WriteString(fmt.Sprintf("  await %s.press(%s);\n", s.Target.locator(), quote(s.Value)))
		case StepExpectVisible:
			b.WriteString(fmt.Sprintf("  await expect(%s).toBeVisible();\n", s.Target.locator()))
		case StepExpectText:
			b.WriteString(fmt.Sprintf("  await expect(%s).toContainText(%s);\n", s.Target.locator(), quote(s.Value)))
		case StepExpectURL:
			b.WriteString(fmt.Sprintf("  await expect(page).toHaveURL(%s);\n", quote(s.Value)))
		}
	}

	b.WriteString("});\n")
	return b.String()
}

// CompileMaestro renders a flow as a Maestro flow.
//
// The device platforms are not wired up yet, but the compiler is written
// alongside the web one to keep the step model honest: a step that cannot be
// expressed on a device does not belong in the model.
func CompileMaestro(f Flow, appID string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("appId: %s\n", appID))
	b.WriteString(fmt.Sprintf("name: %s\n", f.Name))
	b.WriteString("---\n")
	b.WriteString("- launchApp\n")

	for _, s := range f.Steps {
		switch s.Kind {
		case StepGoto:
			b.WriteString(fmt.Sprintf("- openLink: %s\n", s.Value))
		case StepTap:
			b.WriteString(fmt.Sprintf("- tapOn: %s\n", maestroTarget(s.Target)))
		case StepType:
			b.WriteString(fmt.Sprintf("- tapOn: %s\n", maestroTarget(s.Target)))
			b.WriteString(fmt.Sprintf("- inputText: %s\n", quoteYAML(s.Value)))
		case StepPress:
			b.WriteString(fmt.Sprintf("- pressKey: %s\n", s.Value))
		case StepExpectVisible, StepExpectText:
			b.WriteString(fmt.Sprintf("- assertVisible: %s\n", maestroTarget(s.Target)))
		}
	}
	return b.String()
}

func maestroTarget(t Target) string {
	switch {
	case t.TestID != "":
		return fmt.Sprintf("\n    id: %s", quoteYAML(t.TestID))
	case t.Name != "":
		return quoteYAML(t.Name)
	case t.Text != "":
		return quoteYAML(t.Text)
	default:
		return quoteYAML(t.CSS)
	}
}

func quote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`, "\n", `\n`).Replace(s) + "'"
}

func quoteYAML(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", " ").Replace(s) + `"`
}
