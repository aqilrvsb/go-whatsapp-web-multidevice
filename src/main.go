package main

import (
	"embed"
	"os"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/cmd"
)

// Force rebuild: 2025-07-02 v1.2.0-whatsapp-web
// This comment forces Go to recompile with updated embedded files including WhatsApp Web

//go:embed views/index.html
var embedIndex embed.FS

//go:embed views
var embedViews embed.FS

func main() {
	// Set timezone to Malaysia
	os.Setenv("TZ", "Asia/Kuala_Lumpur")
	time.Local, _ = time.LoadLocation("Asia/Kuala_Lumpur")
	
	// Version 1.1.0 with device filter fix
	cmd.Execute(embedIndex, embedViews)
}
