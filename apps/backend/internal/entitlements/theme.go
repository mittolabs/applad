package entitlements

import (
	"regexp"
	"strings"
)

// Theme is how a notice is allowed to look.
//
// It is a DECLARED VOCABULARY, not markup. Whoever supplies a notice picks from
// values core knows how to render; it never sends HTML, CSS or script. Two
// reasons, and the second is the one that matters:
//
//  1. Core owns the rendering, so restyling the console cannot break whoever
//     supplied the notice, and a composed build cannot drift from the design.
//  2. A notice is shown inside every customer's authenticated session. Accepting
//     markup from the plane that can already read every account would turn an
//     operator mistake, or a compromised operator, into stored XSS against every
//     customer. No amount of convenience is worth that.
//
// Everything here is validated on the way in. An invalid value is dropped rather
// than passed to the browser, so the notice degrades to the level's default look
// instead of rendering something unintended.
type Theme struct {
	// Background: a solid colour, or a gradient when GradientTo is set.
	Background    string `json:"background,omitempty"`
	GradientTo    string `json:"gradientTo,omitempty"`
	GradientAngle int    `json:"gradientAngle,omitempty"`

	// Image sits behind the content. https only.
	Image string `json:"image,omitempty"`

	// Effect is a named animation core implements.
	Effect string `json:"effect,omitempty"`

	TextColor   string `json:"textColor,omitempty"`
	AccentColor string `json:"accentColor,omitempty"`

	// Height: compact | normal | tall
	Height string `json:"height,omitempty"`
	// Align: left | center
	Align string `json:"align,omitempty"`

	// Icon is rendered as TEXT (an emoji, typically). Never as markup.
	Icon string `json:"icon,omitempty"`
}

// Effects core can render. A provider naming anything else gets none.
const (
	EffectNone     = "none"
	EffectSnow     = "snow"
	EffectConfetti = "confetti"
	EffectShimmer  = "shimmer"
	EffectPulse    = "pulse"
	EffectTwinkle  = "twinkle"
)

var validEffects = map[string]bool{
	EffectNone: true, EffectSnow: true, EffectConfetti: true,
	EffectShimmer: true, EffectPulse: true, EffectTwinkle: true,
}

var validHeights = map[string]bool{"compact": true, "normal": true, "tall": true}
var validAligns = map[string]bool{"left": true, "center": true}

// hexColor is the only colour syntax accepted. Named colours and arbitrary CSS
// are refused: a value reaching an inline style must not be able to smuggle
// anything else in.
var hexColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

func cleanColor(s string) string {
	s = strings.TrimSpace(s)
	if hexColor.MatchString(s) {
		return s
	}
	return ""
}

// cleanImage accepts https URLs only. http would break the page's security
// context and a data: URI is an inline payload by another name.
func cleanImage(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(s), "https://") {
		return ""
	}
	if strings.ContainsAny(s, "\"'()\\<>") { // cannot break out of url(...)
		return ""
	}
	return s
}

// cleanIcon keeps a short piece of text. Rendered as text by the console, so it
// carries no markup regardless, but length is bounded to keep layout sane.
func cleanIcon(s string) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > 4 {
		return string([]rune(s)[:4])
	}
	return s
}

// sanitise returns the theme with every unusable value removed, and nil when
// nothing survives.
func (t *Theme) sanitise() *Theme {
	if t == nil {
		return nil
	}
	out := Theme{
		Background:  cleanColor(t.Background),
		GradientTo:  cleanColor(t.GradientTo),
		TextColor:   cleanColor(t.TextColor),
		AccentColor: cleanColor(t.AccentColor),
		Image:       cleanImage(t.Image),
		Icon:        cleanIcon(t.Icon),
	}
	if validEffects[t.Effect] && t.Effect != EffectNone {
		out.Effect = t.Effect
	}
	if validHeights[t.Height] {
		out.Height = t.Height
	}
	if validAligns[t.Align] {
		out.Align = t.Align
	}
	if a := t.GradientAngle; a >= 0 && a <= 360 {
		out.GradientAngle = a
	}
	// A gradient needs somewhere to start.
	if out.GradientTo != "" && out.Background == "" {
		out.GradientTo = ""
	}
	if out == (Theme{}) {
		return nil
	}
	return &out
}
