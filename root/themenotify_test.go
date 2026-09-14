package root

import (
	"bytes"
	"testing"
)

func TestStripThemeNotifications(t *testing.T) {
	for _, tc := range []struct {
		name      string
		in        []byte
		wantOut   []byte
		wantDark  bool
		wantFound bool
	}{
		{"nothing to find", []byte("ls -la\r"), []byte("ls -la\r"), false, false},
		{"dark on its own", themeNotifyDark, []byte{}, true, true},
		{"light on its own", themeNotifyLight, []byte{}, false, true},
		{
			"keystrokes either side survive in order",
			bytes.Join([][]byte{[]byte("cd "), themeNotifyLight, []byte("src")}, nil),
			[]byte("cd src"), false, true,
		},
		{
			"the last report wins",
			bytes.Join([][]byte{themeNotifyDark, themeNotifyLight}, nil),
			[]byte{}, false, true,
		},
		{
			"an arrow key is not mistaken for one",
			[]byte("\x1b[A\x1b[B\x1b[C\x1b[D"),
			[]byte("\x1b[A\x1b[B\x1b[C\x1b[D"), false, false,
		},
		{
			"a half sequence is left alone rather than swallowed",
			[]byte("\x1b[?997;"),
			[]byte("\x1b[?997;"), false, false,
		},
		{
			"an unknown report number is left alone",
			[]byte("\x1b[?997;9n"),
			[]byte("\x1b[?997;9n"), false, false,
		},
	} {
		out, dark, found := stripThemeNotifications(tc.in)
		if found != tc.wantFound {
			t.Errorf("%s: found=%v, want %v", tc.name, found, tc.wantFound)
		}
		if found && dark != tc.wantDark {
			t.Errorf("%s: dark=%v, want %v", tc.name, dark, tc.wantDark)
		}
		if !bytes.Equal(out, tc.wantOut) {
			t.Errorf("%s: out=%q, want %q", tc.name, out, tc.wantOut)
		}
	}
}

// The scan runs on every read of stdin, so the common case, which is a keystroke and
// nothing else, must not allocate or rewrite anything.
func TestStripThemeNotificationsLeavesOrdinaryInputUntouched(t *testing.T) {
	in := []byte("some ordinary typing\r\n\x1b[A")
	out, _, found := stripThemeNotifications(in)
	if found {
		t.Fatal("found a notification in plain input")
	}
	if &out[0] != &in[0] {
		t.Error("plain input was copied rather than passed through")
	}
}
