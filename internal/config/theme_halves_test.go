package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTheme drops a theme file in a temp dir and hands back its path.
func writeTheme(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "theme.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing theme: %v", err)
	}
	return path
}

func TestBackgroundIsDark(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want bool
	}{
		{"", true},
		{"dark", true},
		{"light", false},
		{"LIGHT", false},
		{"  light  ", false},
		{"anything else", true},
	} {
		t.Setenv(BackgroundEnv, tc.env)
		if got := BackgroundIsDark(); got != tc.want {
			t.Errorf("%s=%q gave %v, want %v", BackgroundEnv, tc.env, got, tc.want)
		}
	}
}

// A theme written before the tables existed has to keep behaving exactly as it did.
func TestFlatThemeIsUnchangedByHalves(t *testing.T) {
	path := writeTheme(t, `border = "#111111"`+"\n")
	for _, bg := range []string{"dark", "light"} {
		t.Setenv(BackgroundEnv, bg)
		LoadTheme(path)
		if got := Theme().Border; got != "#111111" {
			t.Errorf("on %s background border is %q, want the flat value", bg, got)
		}
		if got := Theme().Accent; got != defaultTheme.Accent {
			t.Errorf("on %s background accent is %q, want the built in default", bg, got)
		}
	}
}

func TestHalfOverridesTheBase(t *testing.T) {
	path := writeTheme(t, `
border = "#base00"
match  = "#basematch"

[dark]
border = "#darkborder"

[light]
border = "#lightborder"
`)

	t.Setenv(BackgroundEnv, "dark")
	LoadTheme(path)
	if got := Theme().Border; got != "#darkborder" {
		t.Errorf("dark border is %q, want the dark half", got)
	}
	if got := Theme().Match; got != "#basematch" {
		t.Errorf("dark match is %q, want the base, which the half leaves alone", got)
	}

	t.Setenv(BackgroundEnv, "light")
	LoadTheme(path)
	if got := Theme().Border; got != "#lightborder" {
		t.Errorf("light border is %q, want the light half", got)
	}
	if got := Theme().Match; got != "#basematch" {
		t.Errorf("light match is %q, want the base, which the half leaves alone", got)
	}
}

// A key present only in the half it does not belong to must not leak across, and a
// key in neither half falls back to the built in default rather than to the other one.
func TestHalvesDoNotLeakIntoEachOther(t *testing.T) {
	path := writeTheme(t, `
[dark]
sel_bg = "#3b3552"

[light]
sel_bg = "#dcdbe1"
text   = "#474556"
`)

	t.Setenv(BackgroundEnv, "dark")
	LoadTheme(path)
	if got := Theme().SelBg; got != "#3b3552" {
		t.Errorf("dark sel_bg is %q", got)
	}
	if got := Theme().Text; got != defaultTheme.Text {
		t.Errorf("dark text is %q, want the default, not the light half's value", got)
	}

	t.Setenv(BackgroundEnv, "light")
	LoadTheme(path)
	if got := Theme().SelBg; got != "#dcdbe1" {
		t.Errorf("light sel_bg is %q", got)
	}
	if got := Theme().Text; got != "#474556" {
		t.Errorf("light text is %q", got)
	}
}

// Reloading is what the file watcher does every second, so switching halves between
// two loads has to land on the new half rather than accumulate both.
func TestReloadSwitchesHalvesCleanly(t *testing.T) {
	path := writeTheme(t, `
[dark]
border = "#darkborder"

[light]
border = "#lightborder"
`)

	t.Setenv(BackgroundEnv, "dark")
	LoadTheme(path)
	t.Setenv(BackgroundEnv, "light")
	LoadTheme(path)
	if got := Theme().Border; got != "#lightborder" {
		t.Errorf("after reloading onto the light half border is %q", got)
	}
}
