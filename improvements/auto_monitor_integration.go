// Add this to your main.go or initialization code to start the 15-minute monitor

package main

import (
    "github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
)

// In your main function or startup code, add:
func startAutoConnectionMonitor() {
    // Get the monitor instance
    monitor := whatsapp.GetAutoConnectionMonitor()
    
    // Start monitoring
    err := monitor.Start()
    if err != nil {
        logrus.Errorf("Failed to start auto connection monitor: %v", err)
    } else {
        logrus.Info("Auto connection monitor started - will check every 15 minutes")
    }
}

// Optional: Add HTTP endpoints to control the monitor
func RegisterMonitorEndpoints(app *fiber.App) {
    // Get monitor status
    app.Get("/api/monitor/status", func(c *fiber.Ctx) error {
        monitor := whatsapp.GetAutoConnectionMonitor()
        status := monitor.GetStatus()
        
        return c.JSON(fiber.Map{
            "status": "success",
            "data": status,
        })
    })
    
    // Force immediate check
    app.Post("/api/monitor/check", func(c *fiber.Ctx) error {
        monitor := whatsapp.GetAutoConnectionMonitor()
        monitor.ForceCheck()
        
        return c.JSON(fiber.Map{
            "status": "success",
            "message": "Device check triggered",
        })
    })
    
    // Stop monitor
    app.Post("/api/monitor/stop", func(c *fiber.Ctx) error {
        monitor := whatsapp.GetAutoConnectionMonitor()
        monitor.Stop()
        
        return c.JSON(fiber.Map{
            "status": "success",
            "message": "Monitor stopped",
        })
    })
    
    // Start monitor
    app.Post("/api/monitor/start", func(c *fiber.Ctx) error {
        monitor := whatsapp.GetAutoConnectionMonitor()
        err := monitor.Start()
        
        if err != nil {
            return c.JSON(fiber.Map{
                "status": "error",
                "message": err.Error(),
            })
        }
        
        return c.JSON(fiber.Map{
            "status": "success",
            "message": "Monitor started",
        })
    })
}