// Package style provides custom application CSS as well as other styling
// utilities.
package style

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rkoesters/xkcd-gtk/internal/log"
	"regexp"
	"strings"
	"sync"
)

const (
	ClassComicContainer = "comic-container"
	ClassLinked         = "linked"
	ClassNoMinWidth     = "no-min-width"

	PaddingComicListButton   = 8
	PaddingPopover           = 10
	PaddingPopoverCompact    = 8
	PaddingPropertiesDialog  = 12
	PaddingUnlinkedButtonBox = 4
)

var (
	cssDataMutex      sync.RWMutex
	cssProvider       *gtk.CssProvider // Protected by cssDataMutex
	loadedCSSDarkMode bool             // Protected by cssDataMutex
)

// InitCSS initializes the application's custom CSS.
func InitCSS(darkMode bool) error {
	var err error

	cssDataMutex.Lock()
	defer cssDataMutex.Unlock()

	cssProvider, err = gtk.CssProviderNew()
	if err != nil {
		return err
	}

	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		return err
	}

	gtk.AddProviderForScreen(screen, cssProvider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

	return loadCSS(cssProvider, darkMode)
}

// UpdateCSS reloads the application CSS if darkMode does not match the
// currently loaded CSS.
func UpdateCSS(darkMode bool) error {
	log.Debugf("UpdateCSS(darkMode=%v)", darkMode)
	cssDataMutex.RLock()
	if darkMode == loadedCSSDarkMode {
		cssDataMutex.RUnlock()
		return nil
	}
	cssDataMutex.RUnlock()

	cssDataMutex.Lock()
	defer cssDataMutex.Unlock()

	return loadCSS(cssProvider, darkMode)
}

func loadCSS(p *gtk.CssProvider, darkMode bool) error {
	loadedCSSDarkMode = darkMode
	if darkMode {
		log.Debug("loading style-dark.css")
		return p.LoadFromData(styleDarkCSS)
	} else {
		log.Debug("loading style.css")
		return p.LoadFromData(styleCSS)
	}
}

var (
	largeToolbarThemesRegexp = regexp.MustCompile(strings.Join([]string{
		"elementary(-x)?",
		"io\\.elementary\\.stylesheet.*",
		"win32",
	}, "|"))

	nonSymbolicIconThemesRegexp = regexp.MustCompile(strings.Join([]string{
		"elementary(-x)?",
		"io\\.elementary\\.stylesheet.*",
	}, "|"))

	unlinkedNavButtonsThemesRegexp = regexp.MustCompile(strings.Join([]string{
		"elementary(-x)?",
		"io\\.elementary\\.stylesheet.*",
	}, "|"))

	compactMenuThemesRegexp = regexp.MustCompile(strings.Join([]string{
		"elementary(-x)?",
		"io\\.elementary\\.stylesheet.*",
	}, "|"))
)

// IsLargeToolbarTheme returns true if we should use large toolbar buttons with
// the given theme.
func IsLargeToolbarTheme(theme string) bool {
	return largeToolbarThemesRegexp.MatchString(theme)
}

// IsSymbolicIconTheme returns true if we should use symbolic icons with the
// given theme.
func IsSymbolicIconTheme(theme string, darkMode bool) bool {
	return darkMode || !nonSymbolicIconThemesRegexp.MatchString(theme)
}

// IsLinkedNavButtonsTheme returns true if we should visually "link" the buttons
// in the navigation button box for the given theme.
func IsLinkedNavButtonsTheme(theme string) bool {
	return !unlinkedNavButtonsThemesRegexp.MatchString(theme)
}

// IsCompactMenuTheme returns true if we should reduce the left and right
// margins of popover-style menus.
func IsCompactMenuTheme(theme string) bool {
	return compactMenuThemesRegexp.MatchString(theme)
}
