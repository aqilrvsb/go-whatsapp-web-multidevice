package main

import (
	"embed"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/cmd"
)

// Force rebuild: 2025-06-24 v1.1.0-fixed
// This comment forces Go to recompile with updated embedded files

//go:embed views/index.html
var embedIndex embed.FS

//go:embed views
var embedViews embed.FS

func main() {
	// Version 1.1.0 with device filter fix
	cmd.Execute(embedIndex, embedViews)
}
