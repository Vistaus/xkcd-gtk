// Package build provides access to compile-time build information.
package build

import (
	"github.com/rkoesters/xkcd-gtk/internal/log"
	"strings"
)

// Comma separated list of key=value pairs.
var data = ""

// Data is a [key]=value pair of compile time flags/data. Init must be called
// before using Data.
var Data map[string]string

// Init initializes the build package by parsing the data provided to it at
// compile time. Init must be called before using Data or calling any other
// function provided by this package.
func Init() {
	log.Debug("build data: ", data)

	Data = parse(data)
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

// AppID returns the RDNN ID of this binary.
func AppID() string {
	id, ok := Data["app-id"]
	if ok {
		return id
	} else {
		return "undefined"
	}
}

// Version returns the version string of this binary.
func Version() string {
	v, ok := Data["version"]
	if ok {
		return v
	} else {
		return "undefined"
	}
}
