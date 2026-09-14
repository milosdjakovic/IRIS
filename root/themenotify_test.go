package root

import (
	"bytes"
	"testing"
)

func TestStripThemeNotifications(t *testing.T) {
	for _, tc := range []struct {
		name        string
		in          []byte
		wantOut     []byte
		wantReports [][]byte
	}{
		{"nothing to find", []byte("ls -la\r"), []byte("ls -la\r"), nil},
		{"dark on its own", themeNotifyDark, []byte{}, [][]byte{themeNotifyDark}},
		{"light on its own", themeNotifyLight, []byte{}, [][]byte{themeNotifyLight}},
		{
			"keystrokes either side survive in order",
			bytes.Join([][]byte{[]byte("cd "), themeNotifyLight, []byte("src")}, nil),
			[]byte("cd src"), [][]byte{themeNotifyLight},
		},
		{
			"every report is handed back in order",
			bytes.Join([][]byte{themeNotifyDark, themeNotifyLight}, nil),
			[]byte{}, [][]byte{themeNotifyDark, themeNotifyLight},
		},
		{
			"an arrow key is not mistaken for one",
			[]byte("\x1b[A\x1b[B\x1b[C\x1b[D"),
			[]byte("\x1b[A\x1b[B\x1b[C\x1b[D"), nil,
		},
		{
			"a half sequence is left alone rather than swallowed",
			[]byte("\x1b[?997;"),
			[]byte("\x1b[?997;"), nil,
		},
		{
			"an unknown report number is left alone",
			[]byte("\x1b[?997;9n"),
			[]byte("\x1b[?997;9n"), nil,
		},
	} {
		out, reports := stripThemeNotifications(tc.in)
		if len(reports) != len(tc.wantReports) {
			t.Errorf("%s: %d reports, want %d", tc.name, len(reports), len(tc.wantReports))
		} else {
			for i := range reports {
				if !bytes.Equal(reports[i], tc.wantReports[i]) {
					t.Errorf("%s: report %d=%q, want %q", tc.name, i, reports[i], tc.wantReports[i])
				}
			}
		}
		if !bytes.Equal(out, tc.wantOut) {
			t.Errorf("%s: out=%q, want %q", tc.name, out, tc.wantOut)
		}
	}
}

// The last report wins, because a terminal that flips twice inside one read is saying
// where it ended up.
func TestReportsDark(t *testing.T) {
	if reportsDark(nil) {
		t.Error("no reports read as dark")
	}
	if !reportsDark([][]byte{themeNotifyLight, themeNotifyDark}) {
		t.Error("light then dark did not read as dark")
	}
	if reportsDark([][]byte{themeNotifyDark, themeNotifyLight}) {
		t.Error("dark then light read as dark")
	}
}

// A program behind iris says whether it wants the reports on its own output stream,
// and the scan has to see that whether or not the switch lands whole in one read.
func TestScanThemeNotifySwitch(t *testing.T) {
	request := []byte(EnableThemeNotifications)
	withdraw := []byte(DisableThemeNotifications)
	for _, tc := range []struct {
		name      string
		carry     []byte
		chunk     []byte
		wantWant  bool
		wantFound bool
	}{
		{"plain output", nil, []byte("hello\x1b[0m"), false, false},
		{"a program asks", nil, append([]byte("\x1b[?1049h"), request...), true, true},
		{"a program withdraws", nil, append(append([]byte{}, withdraw...), []byte("\x1b[?1049l")...), false, true},
		{"the last switch wins", nil, append(append([]byte{}, request...), withdraw...), false, true},
		{"a request split across two reads", request[:3], request[3:], true, true},
		{"a withdrawal split across two reads", withdraw[:6], withdraw[6:], false, true},
		{"a switch inside the chunk beats one on the boundary", request[:3], append(append([]byte{}, request[3:]...), withdraw...), false, true},
		{"the alternate screen alone is not a request", nil, []byte("\x1b[?1049h\x1b[2J"), false, false},
	} {
		want, found := scanThemeNotifySwitch(tc.carry, tc.chunk)
		if found != tc.wantFound || (found && want != tc.wantWant) {
			t.Errorf("%s: got (want=%v, found=%v), want (want=%v, found=%v)", tc.name, want, found, tc.wantWant, tc.wantFound)
		}
	}
}

func TestKeepThemeNotifyCarryHoldsOneByteShortOfASwitch(t *testing.T) {
	carry := keepThemeNotifyCarry([]byte("some longer output " + EnableThemeNotifications))
	if len(carry) != themeNotifyCarryLen {
		t.Fatalf("carry is %d bytes, want %d", len(carry), themeNotifyCarryLen)
	}
	if len([]byte(EnableThemeNotifications)) != themeNotifyCarryLen+1 {
		t.Fatalf("the carry must be exactly one byte short of the switch")
	}
}

// The scan runs on every read of stdin, so the common case, which is a keystroke and
// nothing else, must not allocate or rewrite anything.
func TestStripThemeNotificationsLeavesOrdinaryInputUntouched(t *testing.T) {
	in := []byte("some ordinary typing\r\n\x1b[A")
	out, reports := stripThemeNotifications(in)
	if len(reports) > 0 {
		t.Fatal("found a notification in plain input")
	}
	if &out[0] != &in[0] {
		t.Error("plain input was copied rather than passed through")
	}
}
