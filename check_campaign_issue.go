package main

import (
    "database/sql"
    "fmt"
    "log"
    _ "github.com/lib/pq"
)

func main() {
    // Connect to database
    db, err := sql.Open("postgres", "postgresql://postgres:CNFPbgfjsIVirTuqLMoObNMvoYobDDTU@yamanote.proxy.rlwy.net:49914/railway")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Check device status
    deviceID := "d409cadc-75e2-4004-a789-c2bad0b31393"
    
    fmt.Println("=== DEVICE STATUS CHECK ===")
    var status, platform sql.NullString
    err = db.QueryRow("SELECT status, platform FROM user_devices WHERE id = $1", deviceID).Scan(&status, &platform)
    if err != nil {
        fmt.Printf("Device not found: %v\n", err)
    } else {
        fmt.Printf("Device ID: %s\n", deviceID)
        fmt.Printf("Status: '%s'\n", status.String)
        fmt.Printf("Platform: '%s'\n", platform.String)
    }
    
    // Check all device statuses
    fmt.Println("\n=== ALL DEVICE STATUSES ===")
    rows, _ := db.Query("SELECT DISTINCT status, COUNT(*) FROM user_devices GROUP BY status ORDER BY status")
    defer rows.Close()
    
    for rows.Next() {
        var status sql.NullString
        var count int
        rows.Scan(&status, &count)
        fmt.Printf("Status '%s': %d devices\n", status.String, count)
    }
    
    // Check campaign 59
    fmt.Println("\n=== CAMPAIGN 59 STATUS ===")
    var campaignStatus, campaignTitle sql.NullString
    err = db.QueryRow("SELECT status, title FROM campaigns WHERE id = 59").Scan(&campaignStatus, &campaignTitle)
    if err == nil {
        fmt.Printf("Campaign: %s\n", campaignTitle.String)
        fmt.Printf("Status: %s\n", campaignStatus.String)
    }
    
    // Check broadcast messages
    fmt.Println("\n=== BROADCAST MESSAGES FOR DEVICE ===")
    var pendingCount, queuedCount, skippedCount, sentCount, failedCount int
    db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE device_id = $1 AND status = 'pending'", deviceID).Scan(&pendingCount)
    db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE device_id = $1 AND status = 'queued'", deviceID).Scan(&queuedCount)
    db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE device_id = $1 AND status = 'skipped'", deviceID).Scan(&skippedCount)
    db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE device_id = $1 AND status = 'sent'", deviceID).Scan(&sentCount)
    db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE device_id = $1 AND status = 'failed'", deviceID).Scan(&failedCount)
    
    fmt.Printf("Pending: %d\n", pendingCount)
    fmt.Printf("Queued: %d\n", queuedCount)
    fmt.Printf("Skipped: %d\n", skippedCount)
    fmt.Printf("Sent: %d\n", sentCount)
    fmt.Printf("Failed: %d\n", failedCount)
    
    // Check campaign 59 messages
    fmt.Println("\n=== CAMPAIGN 59 BROADCAST MESSAGES ===")
    var campaign59Count int
    db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = 59").Scan(&campaign59Count)
    fmt.Printf("Total messages for campaign 59: %d\n", campaign59Count)
    
    // Show status breakdown for campaign 59
    rows2, _ := db.Query(`
        SELECT status, COUNT(*), MIN(created_at), MAX(created_at)
        FROM broadcast_messages 
        WHERE campaign_id = 59
        GROUP BY status
        ORDER BY status
    `)
    defer rows2.Close()
    
    fmt.Println("\nStatus breakdown:")
    for rows2.Next() {
        var status string
        var count int
        var minTime, maxTime sql.NullString
        rows2.Scan(&status, &count, &minTime, &maxTime)
        fmt.Printf("- %s: %d messages (created between %s and %s)\n", status, count, minTime.String[:19], maxTime.String[:19])
    }
    
    // Check if messages are being created
    fmt.Println("\n=== RECENT BROADCAST MESSAGES ===")
    rows3, _ := db.Query(`
        SELECT id, device_id, campaign_id, status, error_message, created_at, scheduled_at
        FROM broadcast_messages 
        WHERE campaign_id = 59 OR device_id = $1
        ORDER BY created_at DESC 
        LIMIT 10
    `, deviceID)
    defer rows3.Close()
    
    for rows3.Next() {
        var id, deviceID, status string
        var campaignID sql.NullInt64
        var errorMsg sql.NullString
        var createdAt, scheduledAt string
        
        rows3.Scan(&id, &deviceID, &campaignID, &status, &errorMsg, &createdAt, &scheduledAt)
        fmt.Printf("\nID: %s...\n", id[:8])
        fmt.Printf("  Device: %s\n", deviceID)
        fmt.Printf("  Campaign: %d\n", campaignID.Int64)
        fmt.Printf("  Status: %s\n", status)
        fmt.Printf("  Error: %s\n", errorMsg.String)
        fmt.Printf("  Created: %s\n", createdAt[:19])
        fmt.Printf("  Scheduled: %s\n", scheduledAt[:19])
    }
    
    // Check the exact device status format
    fmt.Println("\n=== CHECKING DEVICE STATUS FORMAT ===")
    var exactStatus string
    err = db.QueryRow("SELECT status FROM user_devices WHERE id = $1", deviceID).Scan(&exactStatus)
    if err == nil {
        fmt.Printf("Exact status value: '%s' (length: %d)\n", exactStatus, len(exactStatus))
        fmt.Printf("Status bytes: %v\n", []byte(exactStatus))
    }
}
