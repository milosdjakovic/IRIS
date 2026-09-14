package config

import (
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// BackgroundEnv carries the terminal's background from the watchdog, which asks
// for it once while it still owns the tty on its own, down to the process that
// draws. It is an environment variable for the same reason ConfigDirEnv is, the
// shell rc starts the real session with a bare `exec iris` and arguments do not
// survive that boundary.
//
// Setting it by hand is supported and skips the query, which is the escape hatch
// for a terminal that answers OSC 11 wrongly or not at all.
const BackgroundEnv = "IRIS_TERM_BACKGROUND"

var defaultTheme = ThemeStyles{
	Border:     "#a277ff",
	Accent:     "#61ffca",
	Muted:      "#6d6a7f",
	Text:       "#edecee",
	TextSel:    "#ffffff",
	Key:        "#a277ff",
	Match:      "#61ffca",
	Desc:       "#9692a8",
	DescSel:    "#edecee",
	SelBg:      "#3d375e",
	SelText:    "#110f18",
	ScrollInfo: "#a277ff",
	GhostText:  "#4B4A4C",
	History:    "#1a2d36",
	HistorySel: "#61ffca",
	Sys:        "#1e1d28",
	SysSel:     "#a277ff",
	Alias:      "#2a2342",
	AliasSel:   "#a277ff",
}

var (
	themeMu     sync.RWMutex
	themeStyles = defaultTheme
)

func Theme() ThemeStyles {
	themeMu.RLock()
	defer themeMu.RUnlock()
	return themeStyles
}

type ThemeStyles struct {
	Border     string `toml:"border"`
	Accent     string `toml:"accent"`
	Muted      string `toml:"muted"`
	Text       string `toml:"text"`
	TextSel    string `toml:"text_sel"`
	Key        string `toml:"key"`
	Match      string `toml:"match"`
	Desc       string `toml:"desc"`
	DescSel    string `toml:"desc_sel"`
	SelBg      string `toml:"sel_bg"`
	SelText    string `toml:"sel_text"`
	ScrollInfo string `toml:"scroll_info"`
	GhostText  string `toml:"ghost_text"`
	Sys        string `toml:"sys"`
	SysSel     string `toml:"sys_sel"`
	History    string `toml:"hist"`
	HistorySel string `toml:"hist_sel"`
	Alias      string `toml:"alias"`
	AliasSel   string `toml:"alias_sel"`
}

// BackgroundIsDark reports whether the terminal is painting a dark background,
// reading only what the watchdog already determined. It never queries anything
// itself: by the time the drawing process runs, the wrapper is relaying stdin and
// a terminal's reply to an OSC 11 query would be read by the wrong reader.
//
// Unset means dark, which is what lipgloss returns when a terminal declines to
// answer, and matches the palette iris shipped as its default for its whole life.
func BackgroundIsDark() bool {
	backgroundMu.RLock()
	told := background
	backgroundMu.RUnlock()
	if told != nil {
		return *told
	}
	return !strings.EqualFold(strings.TrimSpace(os.Getenv(BackgroundEnv)), "light")
}

var (
	backgroundMu sync.RWMutex
	background   *bool
)

// ApplyBackground records an appearance the terminal reported while iris was already
// running and reloads the theme under it, which is how a half follows a terminal that
// changes appearance mid session instead of waiting for the next shell.
//
// It takes precedence over the environment because it is newer. The environment holds
// what the terminal said when this session started, and this holds what it said since.
func ApplyBackground(dark bool) {
	backgroundMu.Lock()
	background = &dark
	backgroundMu.Unlock()

	if path, err := ThemePath(); err == nil {
		LoadTheme(path)
	}
}

// themeFile is theme.toml's shape. The nineteen keys at the top level are the
// base, and the two optional tables override it with whatever that half of the
// terminal's appearance needs. A file with no tables behaves exactly as it did
// before they existed, which is what keeps every theme already written valid.
type themeFile struct {
	ThemeStyles
	Dark  ThemeStyles `toml:"dark"`
	Light ThemeStyles `toml:"light"`
}

// LoadTheme reads theme.toml at filePath
// Missing file - use defaults silently
// Missing/empty fields - fall back to the default value for that field
// [dark] and [light] tables - the matching one is layered over the base
func LoadTheme(filePath string) {
	themeMu.Lock()
	defer themeMu.Unlock()

	themeStyles = defaultTheme

	data, err := os.ReadFile(filePath)
	if err != nil {
		// missing or unreadable file: use defaults
		return
	}

	var t themeFile
	if _, err := toml.Decode(string(data), &t); err != nil {
		return
	}

	applyThemeWithFallback(&themeStyles, t.ThemeStyles, defaultTheme)

	half := t.Dark
	if !BackgroundIsDark() {
		half = t.Light
	}
	// The base is already in place, so anything this half leaves empty keeps what
	// the base put there rather than reverting to the built in default. themeStyles
	// is passed by value, so it is a snapshot taken before the copy begins.
	applyThemeWithFallback(&themeStyles, half, themeStyles)
}

// applyThemeWithFallback copies non-empty string fields from src into dst,
// using def for any field that is empty in src
func applyThemeWithFallback(dst *ThemeStyles, src, def ThemeStyles) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src)
	dv2 := reflect.ValueOf(def)

	for i := range dv.NumField() {
		sf := sv.Field(i).String()
		if sf != "" {
			dv.Field(i).SetString(sf)
		} else {
			dv.Field(i).SetString(dv2.Field(i).String())
		}
	}
}
