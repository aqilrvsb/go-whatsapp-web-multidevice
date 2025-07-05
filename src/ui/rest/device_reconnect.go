package rest

import (
	"context"
	
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// ReconnectDeviceSession attempts to reconnect using existing WhatsApp session
func ReconnectDeviceSession(c *fiber.Ctx) error {
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
	
	// Check if device has phone for reconnection
	if device.Phone == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "NO_SESSION",
			Message: "Device has no previous session. Please scan QR code.",
		})
	}
	
	// First check if already connected in ClientManager
	cm := whatsapp.GetClientManager()
	existingClient, _ := cm.GetClient(deviceID)
	if existingClient != nil && existingClient.IsConnected() {
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
	
	// Try to reconnect using PostgreSQL WhatsApp store
	logrus.Infof("Attempting to reconnect device %s using stored session...", deviceID)
	
	// Initialize WhatsApp store
	ctx := context.Background()
	dbLog := waLog.Stdout("Database", config.WhatsappLogLevel, true)
	container, err := sqlstore.New(ctx, "postgres", config.DBURI, dbLog)
	if err != nil {
		logrus.Errorf("Failed to create store container: %v", err)
		return c.JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to initialize WhatsApp store",
		})
	}
	
	// Try to get all devices and find one matching our phone number
	devices, err := container.GetAllDevices(ctx)
	if err != nil {
		logrus.Errorf("Failed to get devices from store: %v", err)
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "QR_REQUIRED",
			Message: "No stored session found. Please scan QR code.",
		})
	}
	
	// Find device by phone number
	var waDevice *store.Device
	for _, d := range devices {
		if d.ID != nil && d.ID.User == device.Phone {
			waDevice = d
			logrus.Infof("Found stored WhatsApp device for phone %s", device.Phone)
			break
		}
	}
	
	if waDevice == nil {
		logrus.Infof("No stored session found for phone %s", device.Phone)
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "QR_REQUIRED",
			Message: "Session not found in database. Please scan QR code.",
		})
	}
	
	// Create client with stored device
	client := whatsmeow.NewClient(waDevice, waLog.Stdout("Client", config.WhatsappLogLevel, true))
	client.EnableAutoReconnect = true
	client.AutoTrustIdentity = true
	
	// Add event handlers
	client.AddEventHandler(func(evt interface{}) {
		whatsapp.HandleDeviceEvent(context.Background(), deviceID, evt)
	})
	
	// Try to connect
	err = client.Connect()
	if err != nil {
		logrus.Errorf("Failed to connect: %v", err)
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "QR_REQUIRED",
			Message: "Failed to reconnect. Please scan QR code.",
		})
	}
	
	// Check if logged in
	if client.IsLoggedIn() {
		// Register with ClientManager
		cm.AddClient(deviceID, client)
		
		// Update device status
		userRepo.UpdateDeviceStatus(deviceID, "online", device.Phone, client.Store.ID.String())
		
		logrus.Infof("✅ Successfully reconnected device %s", deviceID)
		
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Device reconnected successfully!",
			Results: map[string]interface{}{
				"deviceId": deviceID,
				"phone":    device.Phone,
				"jid":      client.Store.ID.String(),
				"status":   "connected",
			},
		})
	}
	
	// Not logged in, disconnect and request QR
	client.Disconnect()
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "QR_REQUIRED",
		Message: "Session expired. Please scan QR code.",
	})
}
