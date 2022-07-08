// Package build provides access to compile-time build information.
package build

import (
	"github.com/rkoesters/xkcd-gtk/internal/log"
	"strings"
)

// AppID is the reverse domain name notation ID of this application.
const AppID = "com.github.rkoesters.xkcd-gtk"

// Comma separated list of key=value pairs.
var data = ""

// Options is a [key]=value pair of compile time flags. Init must be called
// before using Options.
var Options map[string]string

// Cached result of Options["debug"] == "on" to speed up Debug which can
// potentially be called in very quick succession.
var debugMode bool

// Init initializes the build package by parsing the data provided to it at
// compile time. Init must be called before using Options or calling any other
// function provided by this package.
func Init() {
	Options = parse(data)

	// Cache this comparison to speed up Debug.
	debugMode = Options["debug"] == "on"

	log.Debug("build data: ", data)
}

func parse(data string) map[string]string {
	flags := make(map[string]string)

	if data == "" {
		return flags
	}

	for _, s := range strings.Split(data, ",") {
		pair := strings.SplitN(s, "=", 2)
		if len(pair) != 2 {
			log.Print("error parsing build flag: ", s)
			continue
		}
		flags[pair[0]] = pair[1]
	}

	return flags
}

// Debug returns whether this binary is a debugging build.
func Debug() bool {
	return debugMode
}

// Version returns the version string of this binary.
func Version() string {
	v, ok := Options["version"]
	if ok {
		return v
	} else {
		return "undefined"
	}
}
