package rest

import (
	"context"
	"database/sql"
	"strings"
	"time"
	
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// RefreshDevice attempts to reconnect an existing device session
func RefreshDevice(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	if deviceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Device ID is required",
		})
	}
	
	// Get user from session
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	// Verify device ownership
	device, err := userRepo.GetUserDevice(session.UserID, deviceID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not found",
		})
	}
	
	// Check if device has platform - platform devices cannot be refreshed
	if device.Platform != "" {
		return c.Status(403).JSON(utils.ResponseData{
			Status:  403,
			Code:    "PLATFORM_DEVICE",
			Message: "Platform devices cannot be refreshed",
		})
	}
	
	// Check if device has phone/JID for reconnection
	if device.Phone == "" && device.JID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "NO_SESSION",
			Message: "Device has no previous session. Please scan QR code.",
		})
	}
	
	// Try to get existing client from ClientManager
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(deviceID)
	
	if err == nil && client != nil && client.IsConnected() {
		// Already connected - ensure it's registered
		cm.AddClient(deviceID, client)
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "ALREADY_CONNECTED",
			Message: "Device is already connected",
			Results: map[string]interface{}{
				"deviceId": deviceID,
				"phone":    device.Phone,
				"status":   "connected",
			},
		})
	}
	
	// If client exists but not connected, try to reconnect
	if client != nil && !client.IsConnected() {
		logrus.Infof("Attempting to reconnect existing client for device %s", deviceID)
		err = client.Connect()
		if err == nil {
			// Wait for connection
			connected := false
			for i := 0; i < 10; i++ {
				if client.IsConnected() {
					connected = true
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
			
			if connected {
				// Ensure it's in ClientManager
				cm.AddClient(deviceID, client)
				userRepo.UpdateDeviceStatus(deviceID, "online", device.Phone, device.JID)
				
				return c.JSON(utils.ResponseData{
					Status:  200,
					Code:    "SUCCESS",
					Message: "Device refreshed successfully",
					Results: map[string]interface{}{
						"deviceId": deviceID,
						"phone":    device.Phone,
						"status":   "connected",
					},
				})
			}
		}
		logrus.Warnf("Failed to reconnect existing client: %v", err)
	}
	
	// If we have JID, try full reconnection from database
	if device.JID != "" {
		logrus.Infof("Attempting full reconnection for device %s with JID %s", deviceID, device.JID)
		
		// Check database for session
		db, err := sql.Open("postgres", config.DBURI)
		if err != nil {
			logrus.Errorf("Failed to open database: %v", err)
		} else {
			defer db.Close()
			
			// Query whatsmeow_sessions table
			var sessionData []byte
			err = db.QueryRow(`
				SELECT session 
				FROM whatsmeow_sessions 
				WHERE our_jid = $1
				LIMIT 1
			`, device.JID).Scan(&sessionData)
			
			if err == nil && sessionData != nil {
				// Try to create new client with session
				ctx := context.Background()
				dbLog := waLog.Stdout("Database", config.WhatsappLogLevel, true)
				
				// Convert postgresql:// to postgres://
				dbURI := config.DBURI
				if strings.HasPrefix(dbURI, "postgresql://") {
					dbURI = strings.Replace(dbURI, "postgresql://", "postgres://", 1)
				}
				
				container, err := sqlstore.New(ctx, "postgres", dbURI, dbLog)
				if err == nil {
					jid, err := types.ParseJID(device.JID)
					if err == nil {
						waDevice, err := container.GetDevice(ctx, jid)
						if err == nil && waDevice != nil {
							// Create new client
							newClient := whatsmeow.NewClient(waDevice, waLog.Stdout("Client", config.WhatsappLogLevel, true))
							newClient.EnableAutoReconnect = true
							newClient.AutoTrustIdentity = true
							
							// Add event handlers
							newClient.AddEventHandler(func(evt interface{}) {
								whatsapp.HandleDeviceEvent(context.Background(), deviceID, evt)
							})
							
							// Try to connect
							err = newClient.Connect()
							if err == nil {
								// Wait for connection
								connected := false
								for i := 0; i < 20; i++ {
									if newClient.IsConnected() && newClient.IsLoggedIn() {
										connected = true
										break
									}
									time.Sleep(500 * time.Millisecond)
								}
								
								if connected {
									// Register with ClientManager
									cm.AddClient(deviceID, newClient)
									
									// Update device status
									jidStr := ""
									if newClient.Store.ID != nil {
										jidStr = newClient.Store.ID.String()
									}
									userRepo.UpdateDeviceStatus(deviceID, "online", device.Phone, jidStr)
									
									logrus.Infof("Successfully refreshed device %s via full reconnection", deviceID)
									
									return c.JSON(utils.ResponseData{
										Status:  200,
										Code:    "SUCCESS",
										Message: "Device refreshed successfully!",
										Results: map[string]interface{}{
											"deviceId": deviceID,
											"phone":    device.Phone,
											"jid":      jidStr,
											"status":   "connected",
										},
									})
								}
								
								// Disconnect if not successful
								newClient.Disconnect()
							}
						}
					}
				}
			}
		}
	}
	
	// If all else fails, require QR scan
	logrus.Infof("Device %s needs QR scan for reconnection", deviceID)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "QR_REQUIRED",
		Message: "Device session expired. Please scan QR code to reconnect.",
		Results: map[string]interface{}{
			"deviceId": deviceID,
			"phone":    device.Phone,
			"status":   "disconnected",
		},
	})
}
