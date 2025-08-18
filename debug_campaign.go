package main

import (
    "database/sql"
    "fmt"
    "log"
    "time"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    // Connect to database
    dsn := "admin_aqil:admin_aqil@tcp(159.89.198.71:3306)/admin_railway?parseTime=true&loc=Local"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    fmt.Println("\n=== CAMPAIGN TRIGGER DIAGNOSTIC ===")
    
    // 1. Check database time
    var dbTime time.Time
    var timezone string
    err = db.QueryRow("SELECT NOW(), @@session.time_zone").Scan(&dbTime, &timezone)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Database Time: %s\n", dbTime)
    fmt.Printf("Database Timezone: %s\n", timezone)
    
    // 2. Check pending campaigns for today
    fmt.Println("\n--- Pending Campaigns for Today ---")
    query := `
        SELECT id, title, campaign_date, time_schedule, status, user_id, 
               niche, target_status, scheduled_at
        FROM campaigns 
        WHERE status = 'pending' 
        AND campaign_date = CURDATE()
    `
    
    rows, err := db.Query(query)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    
    campaignFound := false
    var campaignID int
    var userID string
    
    for rows.Next() {
        campaignFound = true
        var id int
        var title, status, user_id, niche, target_status string
        var campaign_date, time_schedule sql.NullString
        var scheduled_at sql.NullTime
        
        err := rows.Scan(&id, &title, &campaign_date, &time_schedule, &status, 
                         &user_id, &niche, &target_status, &scheduled_at)
        if err != nil {
            log.Printf("Error scanning: %v\n", err)
            continue
        }
        
        campaignID = id
        userID = user_id
        
        fmt.Printf("\nCampaign ID: %d\n", id)
        fmt.Printf("Title: %s\n", title)
        fmt.Printf("Date: %s\n", campaign_date.String)
        fmt.Printf("Time: %s\n", time_schedule.String)
        fmt.Printf("Status: %s\n", status)
        fmt.Printf("Niche: %s\n", niche)
        fmt.Printf("Target Status: %s\n", target_status)
        fmt.Printf("Scheduled At: %v\n", scheduled_at)
        
        // Check if it should trigger
        if time_schedule.Valid {
            campaignTime := fmt.Sprintf("%s %s:00", campaign_date.String, time_schedule.String)
            fmt.Printf("Combined DateTime: %s\n", campaignTime)
            
            // Parse and compare
            loc, _ := time.LoadLocation("Local")
            triggerTime, err := time.ParseInLocation("2006-01-02 15:04:05", campaignTime, loc)
            if err == nil {
                fmt.Printf("Should Trigger: %v (Current: %s, Trigger: %s)\n", 
                    time.Now().After(triggerTime), time.Now().Format("15:04:05"), triggerTime.Format("15:04:05"))
            }
        }
    }
    
    if !campaignFound {
        fmt.Println("No pending campaigns found for today")
        return
    }
    
    // 3. Check the exact query used by ProcessCampaigns
    fmt.Println("\n--- Testing ProcessCampaigns Query ---")
    processQuery := `
        SELECT c.id, c.user_id, c.title, c.message, c.niche, 
            COALESCE(c.target_status, 'all') AS target_status, 
            COALESCE(c.image_url, '') AS image_url, c.min_delay_seconds, c.max_delay_seconds,
            c.campaign_date, c.time_schedule
        FROM campaigns c
        WHERE c.status = 'pending'
        AND (
            (c.scheduled_at IS NOT NULL AND c.scheduled_at <= CURRENT_TIMESTAMP)
            OR
            (c.scheduled_at IS NULL AND 
             STR_TO_DATE(CONCAT(c.campaign_date, ' ', COALESCE(c.time_schedule, '00:00:00')), '%Y-%m-%d %H:%i:%s') <= CONVERT_TZ(NOW(), @@session.time_zone, 'Asia/Kuala_Lumpur'))
        )
    `
    
    rows2, err := db.Query(processQuery)
    if err != nil {
        fmt.Printf("ProcessCampaigns query error: %v\n", err)
    } else {
        defer rows2.Close()
        count := 0
        for rows2.Next() {
            count++
        }
        fmt.Printf("Campaigns ready to trigger: %d\n", count)
    }
    
    // 4. Check if there are leads for the campaign
    if campaignFound {
        fmt.Printf("\n--- Checking Leads for User %s ---\n", userID)
        
        var leadCount int
        err = db.QueryRow("SELECT COUNT(*) FROM leads WHERE user_id = ?", userID).Scan(&leadCount)
        if err == nil {
            fmt.Printf("Total leads: %d\n", leadCount)
        }
        
        // Check leads by niche
        var nicheleadCount int
        err = db.QueryRow("SELECT COUNT(*) FROM leads WHERE user_id = ? AND niche = ?", userID, "").Scan(&nicheleadCount)
        if err == nil {
            fmt.Printf("Leads matching campaign niche: %d\n", nicheleadCount)
        }
        
        // Check connected devices
        fmt.Println("\n--- Checking Connected Devices ---")
        deviceQuery := `
            SELECT id, device_name, status, phone 
            FROM user_devices 
            WHERE user_id = ? 
            AND (status = 'connected' OR status = 'online')
        `
        
        deviceRows, err := db.Query(deviceQuery, userID)
        if err == nil {
            defer deviceRows.Close()
            deviceCount := 0
            for deviceRows.Next() {
                deviceCount++
                var id, name, status, phone string
                deviceRows.Scan(&id, &name, &status, &phone)
                fmt.Printf("Device: %s (%s) - Status: %s\n", name, phone, status)
            }
            fmt.Printf("Total connected devices: %d\n", deviceCount)
        }
        
        // 5. Check if there are any broadcast messages for this campaign
        fmt.Printf("\n--- Checking Broadcast Messages for Campaign %d ---\n", campaignID)
        var broadcastCount int
        err = db.QueryRow("SELECT COUNT(*) FROM broadcast_messages WHERE campaign_id = ?", campaignID).Scan(&broadcastCount)
        if err == nil {
            fmt.Printf("Broadcast messages created: %d\n", broadcastCount)
        }
    }
    
    // 6. Test timezone conversion
    fmt.Println("\n--- Testing Timezone Conversion ---")
    var malaysiaTime sql.NullString
    err = db.QueryRow("SELECT CONVERT_TZ(NOW(), @@session.time_zone, 'Asia/Kuala_Lumpur')").Scan(&malaysiaTime)
    if err != nil {
        fmt.Printf("Timezone conversion error: %v\n", err)
        fmt.Println("This might be why campaigns aren't triggering!")
    } else {
        fmt.Printf("Malaysia time: %v\n", malaysiaTime)
    }
}
