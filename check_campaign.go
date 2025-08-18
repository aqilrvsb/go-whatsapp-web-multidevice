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

    // First, let's check current time settings
    fmt.Println("\n=== DATABASE TIME SETTINGS ===")
    var dbTime, dbTimezone string
    err = db.QueryRow("SELECT NOW(), @@session.time_zone").Scan(&dbTime, &dbTimezone)
    if err == nil {
        fmt.Printf("Database NOW(): %s\n", dbTime)
        fmt.Printf("Database Timezone: %s\n", dbTimezone)
    }

    