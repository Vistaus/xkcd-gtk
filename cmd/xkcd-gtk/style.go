package main

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
)

const css = `
@define-color colorPrimary #96a8c8;
@define-color textColorPrimary #1a1a1a;

.comic-container > .frame {
	background-color: #ffffff;
}
`

// LoadCSS provides the application's custom CSS to GTK.
func (a *Application) LoadCSS() {
	provider, err := gtk.CssProviderNew()
	if err != nil {
		log.Print(err)
		return
	}
	provider.LoadFromData(css)

	screen, err := gdk.ScreenGetDefault()
	if err != nil {
		log.Print(err)
		return
	}

	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
