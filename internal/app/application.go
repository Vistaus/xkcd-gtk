// Package app implements the xkcd-gtk application.
package app

import (
	"errors"
	"flag"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/rkoesters/xkcd-gtk/internal/bookmarks"
	"github.com/rkoesters/xkcd-gtk/internal/cache"
	"github.com/rkoesters/xkcd-gtk/internal/log"
	"github.com/rkoesters/xkcd-gtk/internal/paths"
	"github.com/rkoesters/xkcd-gtk/internal/search"
	"github.com/rkoesters/xkcd-gtk/internal/settings"
	"github.com/rkoesters/xkcd-gtk/internal/style"
	"github.com/rkoesters/xkcd-gtk/internal/widget"
)

var (
	forceAppMenu = flag.Bool("force-app-menu", false, "Always set an app menu.")
)

// Name is the user-visible name of this application.
func Name() string { return widget.AppName() }

// Application holds onto our GTK representation of our application.
type Application struct {
	*gtk.Application

	gtkSettings *gtk.Settings
	actions     map[string]*glib.SimpleAction

	aboutDialog      *gtk.AboutDialog
	shortcutsWindow  *gtk.ShortcutsWindow
	cacheWindow      *widget.CacheWindow
	cacheWindowMutex sync.RWMutex

	settings  settings.Settings
	bookmarks bookmarks.List
}

// New creates an instance of our GTK Application.
func New(appID string, flags glib.ApplicationFlags) (*Application, error) {
	super, err := gtk.ApplicationNew(appID, flags)
	if err != nil {
		return nil, err
	}
	app := Application{
		Application: super,

		actions: make(map[string]*glib.SimpleAction),
	}

	registerAction := func(name string, fn interface{}) {
		action := glib.SimpleActionNew(name, nil)
		action.Connect("activate", fn)

		app.actions[name] = action
		app.AddAction(action)
	}

	registerAction("new-window", app.Activate)
	registerAction("open-about-xkcd", app.OpenAboutXKCD)
	registerAction("open-blog", app.OpenBlog)
	registerAction("open-books", app.OpenBooks)
	registerAction("open-what-if", app.OpenWhatIf)
	registerAction("quit", app.PleaseQuit)
	registerAction("show-about", app.ShowAbout)
	registerAction("show-cache", app.ShowCache)
	registerAction("show-shortcuts", app.ShowShortcuts)
	registerAction("toggle-dark-mode", app.ToggleDarkMode)

	// Initialize our application accelerators.
	app.SetAccelsForAction("app.new-window", []string{"<Control>n"})
	app.SetAccelsForAction("app.quit", []string{"<Control>q"})
	app.SetAccelsForAction("app.show-shortcuts", []string{"<Control>question"})
	app.SetAccelsForAction("app.toggle-dark-mode", []string{"<Control>t"})

	// Connect application signal handlers.
	app.Connect("startup", app.Startup)
	app.Connect("shutdown", app.Shutdown)
	app.Connect("activate", app.Activate)

	return &app, nil
}

// GtkApplication returns the underlying gtk.Application struct.
func (app *Application) GtkApplication() *gtk.Application {
	return app.Application
}

// Startup is called when the "startup" signal is emitted.
func (app *Application) Startup() {
	var err error

	app.gtkSettings, err = gtk.SettingsGetDefault()
	if err != nil {
		log.Fatal("error calling gtk.SettingsGetDefault(): ", err)
	}

	app.LoadSettings()

	err = app.SetupAppMenu()
	if err != nil {
		log.Fatal("error creating app menu: ", err)
	}

	style.InitCSS(app.DarkMode())
	app.gtkSettings.Connect("notify::gtk-application-prefer-dark-theme", app.DarkModeChanged)

	app.LoadBookmarks()
	app.SetupCache()

	// ApplicationWindow.RandomComic() would like a seeded PRNG.
	rand.Seed(time.Now().Unix())
}

// Shutdown is called when the "shutdown" signal is emitted.
func (app *Application) Shutdown() {
	app.SaveSettings()
	app.SaveBookmarks()
	app.CloseCache()
}

// PrefersAppMenu is a wrapper around gtk.Application.PrefersAppMenu().
func (app *Application) PrefersAppMenu() bool {
	return app.Application.PrefersAppMenu() || *forceAppMenu
}

// SetupAppMenu creates an AppMenu if the environment wants it.
func (app *Application) SetupAppMenu() error {
	if !app.PrefersAppMenu() {
		return nil
	}
	menu, err := widget.NewAppMenu()
	if err != nil {
		return err
	}
	app.SetAppMenu(menu)
	return nil
}

// SetupCache initializes the comic cache and the search index.
func (app *Application) SetupCache() {
	log.Debug("SetupCache() start")
	defer log.Debug("SetupCache() end")

	log.Debug("calling cache.Init()")
	err := cache.Init(search.Index)
	if err != nil {
		log.Fatal("error initializing comic cache: ", err)
	}

	log.Debug("calling search.Init()")
	err = search.Init()
	if err != nil {
		log.Fatal("error initializing search index: ", err)
	}

	// Asynchronously fill the comic metadata cache and search index.
	log.Debug("calling cache.DownloadAllComicMetadata()")
	go cache.DownloadAllComicMetadata(app.CacheWindowVRW)
}

// CloseCache closes the search index and comic cache.
func (app *Application) CloseCache() {
	log.Debug("CloseCache() start")
	defer log.Debug("CloseCache() end")

	log.Debug("calling search.Close()")
	err := search.Close()
	if err != nil {
		log.Print("error closing search index: ", err)
	}

	log.Debug("calling cache.Close()")
	err = cache.Close()
	if err != nil {
		log.Print("error closing comic cache: ", err)
	}
}

// Activate creates and presents a new window to the user.
func (app *Application) Activate() {
	win, err := widget.NewApplicationWindow(app)
	if err != nil {
		log.Print("error creating window: ", err)
		return
	}
	win.Present()
}

// DarkModeChanged is called when gtk-application-prefer-dark-theme is changed.
func (app *Application) DarkModeChanged() {
	darkMode := app.DarkMode()
	log.Debugf("DarkModeChanged() -> %v", darkMode)
	err := style.UpdateCSS(darkMode)
	if err != nil {
		log.Printf("error calling style.UpdateCSS(darkMode=%v) -> %v", darkMode, err)
	}
}

// ToggleDarkMode toggles the value of "gtk-application-prefer-dark-theme".
func (app *Application) ToggleDarkMode() {
	app.SetDarkMode(!app.DarkMode())
}

// SetDarkMode sets the value of "gtk-application-prefer-dark-theme" to the
// darkMode argument.
func (app *Application) SetDarkMode(darkMode bool) {
	log.Debugf("SetDarkMode(darkMode=%v)", darkMode)
	// Change the dark mode setting in one of the next iterations of the event
	// loop (i.e. do not block) so that the style does not change in the middle
	// of any ongoing animations (e.g. a switch toggling or a menu closing).
	go glib.IdleAdd(func() {
		log.Debugf("SetDarkMode(darkMode=%v).func()", darkMode)
		// Setting 'gtk-application-prefer-dark-theme' will trigger a call to
		// win.DrawComic which will call app.DarkMode again, which will then
		// update app.settings.DarkMode (which effectively serves as a cache of
		// 'gtk-application-prefer-dark-theme').
		err := app.gtkSettings.SetProperty("gtk-application-prefer-dark-theme", darkMode)
		if err != nil {
			log.Print("error setting dark mode state: ", err)
		}
	})
}

// DarkMode returns whether the application has dark mode enabled.
func (app *Application) DarkMode() bool {
	// Ask GTK whether it is using a dark theme.
	darkModeIface, err := app.gtkSettings.GetProperty("gtk-application-prefer-dark-theme")
	if err != nil {
		log.Print("error getting dark mode state: ", err)
		return app.settings.DarkMode
	}

	darkMode, ok := darkModeIface.(bool)
	if !ok {
		log.Print("failed to interpret dark mode state")
		return app.settings.DarkMode
	}

	// Sync app.settings.DarkMode with the value of
	// 'gtk-application-prefer-dark-theme'.
	app.settings.DarkMode = darkMode

	return darkMode
}

// ConnectDarkModeChanged connects the given signal handler to the dark mode GTK
// signal.
func (app *Application) ConnectDarkModeChanged(f interface{}) glib.SignalHandle {
	return app.gtkSettings.Connect("notify::gtk-application-prefer-dark-theme", f)
}

// PleaseQuit closes all windows so that the application will exit. Main
// functional difference with Quit is that PleaseQuit gives windows the
// opportunity to save state before the application exits.
func (app *Application) PleaseQuit() {
	windows := app.GetWindows()
	windows.Foreach(func(iw interface{}) {
		win, ok := iw.(*gtk.Window)
		if !ok {
			log.Print("error converting window to gtk.Window")
			return
		}
		if win == nil {
			log.Print("window is nil")
			return
		}
		win.Close()
	})
	// Add Quit to end of event queue to give windows time to save state.
	glib.IdleAddPriority(glib.PRIORITY_LOW, app.Quit)
}

// LoadSettings tries to load our settings from disk.
func (app *Application) LoadSettings() {
	log.Debug("LoadSettings() start")
	defer log.Debug("LoadSettings() end")

	paths.CheckForMisplacedSettings()

	// Read settings from disk.
	err := app.settings.ReadFile(paths.Settings())
	if err != nil {
		log.Print("error reading app settings: ", err)
	}

	// Apply Dark Mode setting.
	err = app.gtkSettings.SetProperty("gtk-application-prefer-dark-theme", app.settings.DarkMode)
	if err != nil {
		log.Print("error setting dark mode state: ", err)
	}
}

// SaveSettings tries to save our settings to disk.
func (app *Application) SaveSettings() {
	log.Debug("SaveSettings() start")
	defer log.Debug("SaveSettings() end")

	err := paths.EnsureConfigDir()
	if err != nil {
		log.Printf("error saving settings: %v", err)
	}

	err = app.settings.WriteFile(paths.Settings())
	if err != nil {
		log.Printf("error saving settings: %v", err)
	}
}

// LoadBookmarks tries to load our bookmarks from disk.
func (app *Application) LoadBookmarks() {
	log.Debug("LoadBookmarks() start")
	defer log.Debug("LoadBookmarks() end")

	paths.CheckForMisplacedBookmarks()

	app.bookmarks = bookmarks.New()
	err := app.bookmarks.ReadFile(paths.Bookmarks())
	if err != nil {
		log.Print("error reading bookmarks: ", err)
	}
}

// SaveBookmarks tries to save our bookmarks to disk.
func (app *Application) SaveBookmarks() {
	log.Debug("SaveBookmarks() start")
	defer log.Debug("SaveBookmarks() end")

	err := paths.EnsureDataDir()
	if err != nil {
		log.Printf("error saving bookmarks: %v", err)
	}

	err = app.bookmarks.WriteFile(paths.Bookmarks())
	if err != nil {
		log.Printf("error saving bookmarks: %v", err)
	}
}

// BookmarksList returns a pointer to the app's list of bookmarks.
func (app *Application) BookmarksList() *bookmarks.List {
	return &app.bookmarks
}

// ShowShortcuts shows a shortcuts window to the user.
func (app *Application) ShowShortcuts() {
	if app.shortcutsWindow == nil {
		sw, err := widget.NewShortcutsWindow(app.RemoveWindow)
		if err != nil {
			log.Print("error creating shortcuts window: ", err)
			return
		}
		app.shortcutsWindow = sw
	}
	app.AddWindow(app.shortcutsWindow)
	app.shortcutsWindow.Present()
}

// ShowAbout shows our application info to the user.
func (app *Application) ShowAbout() {
	if app.aboutDialog == nil {
		ad, err := widget.NewAboutDialog(app.RemoveWindow)
		if err != nil {
			log.Print("error creating about dialog: ", err)
			return
		}
		app.aboutDialog = ad
	}
	app.AddWindow(app.aboutDialog)
	app.aboutDialog.Present()
}

// ShowCache shows the cache management window to the user.
func (app *Application) ShowCache() {
	if app.cacheWindow == nil {
		app.cacheWindowMutex.Lock()
		cw, err := widget.NewCacheWindow(app)
		if err != nil {
			log.Print("error creating cache window: ", err)
			app.cacheWindowMutex.Unlock()
			return
		}
		app.cacheWindow = cw
		app.cacheWindowMutex.Unlock()
	}
	app.AddWindow(app.cacheWindow)
	app.cacheWindow.Present()
}

func (app *Application) CacheWindowVR() cache.ViewRefresher {
	app.cacheWindowMutex.RLock()
	defer app.cacheWindowMutex.RUnlock()
	return app.cacheWindow
}

func (app *Application) CacheWindowVRW() cache.ViewRefreshWither {
	app.cacheWindowMutex.RLock()
	defer app.cacheWindowMutex.RUnlock()
	return app.cacheWindow
}

// GtkTheme returns the name of the GTK theme that the application should use.
func (app *Application) GtkTheme() (string, error) {
	theme := os.Getenv("GTK_THEME")
	if theme != "" {
		return theme, nil
	}
	themeIface, err := app.gtkSettings.GetProperty("gtk-theme-name")
	if err != nil {
		return "", err
	}
	var ok bool
	theme, ok = themeIface.(string)
	if !ok {
		return "", errors.New("error converting gtk-theme-name to a string")
	}
	return theme, nil
}
