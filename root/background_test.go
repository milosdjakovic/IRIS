package root

import (
	"image/color"
	"testing"

	"github.com/versenilvis/iris/internal/config"
)

func TestIsDarkBackground(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    color.Color
		want bool
	}{
		{"aura dark page", color.RGBA{0x15, 0x14, 0x1b, 0xff}, true},
		{"aura light page", color.RGBA{0xf8, 0xf7, 0xfb, 0xff}, false},
		{"black", color.RGBA{0, 0, 0, 0xff}, true},
		{"white", color.RGBA{0xff, 0xff, 0xff, 0xff}, false},
	} {
		if got := isDarkBackground(tc.c); got != tc.want {
			t.Errorf("%s gave %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A fresh answer has to replace an inherited one rather than sit beside it. This is the
// bug the environment variable had on its own: it travels to the shell, so a new iris
// started from that shell inherited a decision made about a terminal that may since have
// changed appearance, and the stale value won.
func TestWithBackgroundReplacesAnInheritedAnswer(t *testing.T) {
	env := []string{"PATH=/usr/bin", config.BackgroundEnv + "=light", "TERM=xterm"}
	got := withBackground(env, "dark")

	seen := 0
	for _, kv := range got {
		if len(kv) > len(config.BackgroundEnv) && kv[:len(config.BackgroundEnv)+1] == config.BackgroundEnv+"=" {
			seen++
			if kv != config.BackgroundEnv+"=dark" {
				t.Errorf("kept %q, want the fresh answer", kv)
			}
		}
	}
	if seen != 1 {
		t.Errorf("found %d entries for %s, want exactly one", seen, config.BackgroundEnv)
	}
	if len(got) != len(env) {
		t.Errorf("environment went from %d entries to %d", len(env), len(got))
	}
}

func TestWithBackgroundLeavesEverythingElseAlone(t *testing.T) {
	env := []string{"PATH=/usr/bin", "TERM=xterm"}
	got := withBackground(env, "light")
	if len(got) != 3 {
		t.Fatalf("got %d entries, want the two originals plus one", len(got))
	}
	if got[0] != "PATH=/usr/bin" || got[1] != "TERM=xterm" {
		t.Errorf("originals were disturbed: %v", got[:2])
	}
}
