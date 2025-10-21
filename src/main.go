package main

import (
	"embed"
	"os"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/cmd"
)

// Force rebuild: 2025-10-19 v1.3.2-default-today-fix
// CRITICAL FIX: All 3 pages now default to TODAY when no date filter
// All pages use scheduled_at column and show MATCHING numbers
// Detail Sequences, Report NEW, Progress NEW = TALLY NOW!

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
