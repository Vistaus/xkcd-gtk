// Package paths provides information on where to find and store files.
package paths

import (
	"os"
	"path/filepath"
)

var b builder

// Init initializes the paths package. Must be called before calling other
// package functions.
func Init(appID string) {
	b = builder{
		appID: appID,
	}

	// Fix for a bug where we called `b.ConfigDir()` in `DataDir()`.
	old := filepath.Join(ConfigDir(), "bookmarks")
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		os.Rename(old, filepath.Join(DataDir(), "bookmarks"))
	}
}

// CacheDir returns the path to our app's cache directory.
func CacheDir() string {
	return b.CacheDir()
}

// ConfigDir returns the path to our app's user configuration directory.
func ConfigDir() string {
	return b.ConfigDir()
}

// DataDir returns the path to our app's user data directory.
func DataDir() string {
	return b.DataDir()
}

// LocaleDir returns the path to the system locale directory.
func LocaleDir() string {
	return b.LocaleDir()
}
