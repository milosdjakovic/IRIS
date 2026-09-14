package root

import "bytes"

// A terminal that has been asked for theme change notifications, DEC private mode
// 2031, reports a switch as CSI ? 997 ; 1 n for dark and ; 2 n for light. It arrives
// on the same stream as keystrokes, unprompted, so it has to be taken out before the
// bytes reach the shell or the sequence is printed onto the prompt as text.
var (
	themeNotifyPrefix = []byte("\x1b[?997;")
	themeNotifyDark   = []byte("\x1b[?997;1n")
	themeNotifyLight  = []byte("\x1b[?997;2n")
)

// EnableThemeNotifications asks the terminal to report appearance changes, and
// DisableThemeNotifications withdraws the request. A terminal that does not know the
// mode ignores both, which is why neither is guarded by a capability check.
const (
	EnableThemeNotifications  = "\x1b[?2031h"
	DisableThemeNotifications = "\x1b[?2031l"
)

// stripThemeNotifications removes every theme change notification from a chunk of
// input and hands back the removed sequences, in order, so the caller can act on the
// last one and relay them to a program that asked for them. The rest of the chunk is
// returned untouched and in order, so ordinary keystrokes are unaffected.
//
// Only whole sequences are recognised. A notification split across two reads is
// left alone rather than held back, because holding back a partial escape sequence
// would delay the arrow keys, which begin the same way, and a terminal sends this
// nine byte report as one write of its own rather than interleaved with typing.
func stripThemeNotifications(chunk []byte) (out []byte, reports [][]byte) {
	if !bytes.Contains(chunk, themeNotifyPrefix) {
		return chunk, nil
	}
	out = make([]byte, 0, len(chunk))
	for i := 0; i < len(chunk); {
		switch {
		case bytes.HasPrefix(chunk[i:], themeNotifyDark):
			reports = append(reports, themeNotifyDark)
			i += len(themeNotifyDark)
		case bytes.HasPrefix(chunk[i:], themeNotifyLight):
			reports = append(reports, themeNotifyLight)
			i += len(themeNotifyLight)
		default:
			out = append(out, chunk[i])
			i++
		}
	}
	return out, reports
}

// reportsDark says which appearance the last of a run of reports named. The last one
// wins because a terminal that flips twice inside one read is telling you where it
// ended up.
func reportsDark(reports [][]byte) bool {
	return len(reports) > 0 && bytes.Equal(reports[len(reports)-1], themeNotifyDark)
}

// The program behind iris may ask for the same reports, and it says so on its output
// stream with the same mode switch iris sends. Iris owns the terminal's copy of the
// mode, since it always wants the reports for its own theme, so the child's request is
// not a matter for the terminal but a matter for iris. It decides whether a report,
// once taken out of the input, is also written on to the child.
//
// Without this, anything running behind iris that follows the terminal's appearance
// is deaf to it. A multiplexer is the worst case, since every program inside it asks
// the multiplexer rather than the terminal, so one lost report freezes a whole tree.
var (
	themeNotifyRequest  = []byte(EnableThemeNotifications)
	themeNotifyWithdraw = []byte(DisableThemeNotifications)
)

// themeNotifyCarryLen is one byte short of the request, so a switch split across
// two PTY reads is still seen whole, the same shape as the alternate screen scan.
const themeNotifyCarryLen = 7

// lastThemeNotifySwitch reports the final request or withdrawal in chunk.
func lastThemeNotifySwitch(chunk []byte) (want bool, found bool) {
	on := bytes.LastIndex(chunk, themeNotifyRequest)
	off := bytes.LastIndex(chunk, themeNotifyWithdraw)
	if on < 0 && off < 0 {
		return false, false
	}
	return on > off, true
}

// scanThemeNotifySwitch reports the final request or withdrawal in chunk, also
// looking at the boundary with the previous read so a sequence split across two
// reads is not missed.
func scanThemeNotifySwitch(carry, chunk []byte) (want bool, found bool) {
	if len(carry) > 0 {
		head := chunk
		if len(head) > themeNotifyCarryLen {
			head = head[:themeNotifyCarryLen]
		}
		joined := make([]byte, 0, len(carry)+len(head))
		joined = append(append(joined, carry...), head...)
		if w, ok := lastThemeNotifySwitch(joined); ok {
			want, found = w, true
		}
	}
	// anything wholly inside chunk comes after the boundary, so it wins
	if w, ok := lastThemeNotifySwitch(chunk); ok {
		want, found = w, true
	}
	return want, found
}

func keepThemeNotifyCarry(chunk []byte) []byte {
	if len(chunk) > themeNotifyCarryLen {
		chunk = chunk[len(chunk)-themeNotifyCarryLen:]
	}
	return append([]byte(nil), chunk...)
}
