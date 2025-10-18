package main

import (
	"embed"
	"os"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/cmd"
)

// Force rebuild: 2025-10-18 v1.3.0-new-accurate-pages
// NEW FEATURE: Created 2 brand new pages (Report & Progress) with accurate data matching Detail Sequences
// Both new pages use EXACT same SQL query as Detail Sequences to ensure 100% data consistency

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
