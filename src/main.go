package main

import (
	"embed"
	"os"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/cmd"
)

// Force rebuild: 2025-10-19 v1.3.1-scheduled-at-fix
// CRITICAL FIX: Standardized all pages to use scheduled_at for date filtering (not created_at)
// All 3 pages now show matching numbers: Detail Sequences, Report NEW, Progress NEW
// scheduled_at = actual send date, created_at = record creation date

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
