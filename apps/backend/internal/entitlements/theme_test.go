package entitlements

import "testing"

// A theme reaches inline styles in every customer's authenticated console, so
// anything that is not a value core recognises must not survive.
func TestSanitiseRejectsAnythingNotDeclared(t *testing.T) {
	cases := []struct {
		name  string
		in    Theme
		check func(*Theme) bool
	}{
		{
			"css injection through a colour",
			Theme{Background: "red; background-image: url(javascript:alert(1))"},
			func(o *Theme) bool { return o == nil },
		},
		{"named colour is not hex", Theme{Background: "rebeccapurple"}, func(o *Theme) bool { return o == nil }},
		{
			"url() breakout in an image",
			Theme{Image: "https://x.test/a.png\") ; background: url(evil"},
			func(o *Theme) bool { return o == nil },
		},
		{"data uri is an inline payload", Theme{Image: "data:image/svg+xml;base64,PHN2Zz4="}, func(o *Theme) bool { return o == nil }},
		{"plain http is refused", Theme{Image: "http://x.test/a.png"}, func(o *Theme) bool { return o == nil }},
		{"unknown effect is dropped", Theme{Effect: "rickroll"}, func(o *Theme) bool { return o == nil }},
		{"unknown height is dropped", Theme{Height: "enormous"}, func(o *Theme) bool { return o == nil }},
		{"angle out of range is dropped", Theme{GradientAngle: 900}, func(o *Theme) bool { return o == nil }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.in.sanitise(); !c.check(got) {
				t.Fatalf("value survived sanitising: %+v", got)
			}
		})
	}
}

func TestSanitiseKeepsDeclaredValues(t *testing.T) {
	in := Theme{
		Background:    "#0b3d2e",
		GradientTo:    "#8b0000",
		GradientAngle: 135,
		Image:         "https://cdn.example.com/snow.png",
		Effect:        EffectSnow,
		TextColor:     "#ffffff",
		AccentColor:   "#e0a030",
		Height:        "tall",
		Align:         "center",
		Icon:          "🎄",
	}
	out := in.sanitise()
	if out == nil {
		t.Fatal("a fully valid theme was dropped")
	}
	if out.Background != "#0b3d2e" || out.GradientTo != "#8b0000" || out.GradientAngle != 135 {
		t.Errorf("gradient not preserved: %+v", out)
	}
	if out.Effect != EffectSnow || out.Height != "tall" || out.Align != "center" {
		t.Errorf("presentation not preserved: %+v", out)
	}
	if out.Icon != "🎄" || out.Image == "" {
		t.Errorf("icon/image not preserved: %+v", out)
	}
}

// A gradient with no starting colour is not a gradient.
func TestGradientNeedsABase(t *testing.T) {
	th := Theme{GradientTo: "#ffffff"}
	out := th.sanitise()
	if out != nil && out.GradientTo != "" {
		t.Fatalf("dangling gradient survived: %+v", out)
	}
}

func TestIconIsLengthBounded(t *testing.T) {
	th := Theme{Icon: "🎄🎄🎄🎄🎄🎄🎄🎄"}
	out := th.sanitise()
	if out == nil {
		t.Fatal("icon dropped entirely")
	}
	if len([]rune(out.Icon)) > 4 {
		t.Fatalf("icon not bounded: %q", out.Icon)
	}
}

// Sanitising runs as part of normalise, so nothing reaches a client unchecked.
func TestNormaliseSanitisesThemes(t *testing.T) {
	d := normalise(Document{Notices: []Notice{{
		ID: "n1", Region: RegionAppTop, Title: "x",
		Theme: &Theme{Background: "expression(alert(1))", Effect: "nope"},
	}}})
	if len(d.Notices) != 1 {
		t.Fatalf("notice dropped: %+v", d.Notices)
	}
	if d.Notices[0].Theme != nil {
		t.Fatalf("unsafe theme reached the client: %+v", d.Notices[0].Theme)
	}
}
