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
// input and reports the appearance the last one named. The rest of the chunk is
// returned untouched and in order, so ordinary keystrokes are unaffected.
//
// Only whole sequences are recognised. A notification split across two reads is
// left alone rather than held back, because holding back a partial escape sequence
// would delay the arrow keys, which begin the same way, and a terminal sends this
// nine byte report as one write of its own rather than interleaved with typing.
func stripThemeNotifications(chunk []byte) (out []byte, dark bool, found bool) {
	if !bytes.Contains(chunk, themeNotifyPrefix) {
		return chunk, false, false
	}
	out = make([]byte, 0, len(chunk))
	for i := 0; i < len(chunk); {
		switch {
		case bytes.HasPrefix(chunk[i:], themeNotifyDark):
			dark, found = true, true
			i += len(themeNotifyDark)
		case bytes.HasPrefix(chunk[i:], themeNotifyLight):
			dark, found = false, true
			i += len(themeNotifyLight)
		default:
			out = append(out, chunk[i])
			i++
		}
	}
	return out, dark, found
}
