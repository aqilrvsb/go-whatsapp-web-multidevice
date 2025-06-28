package rest

import (
	"fmt"
	"path/filepath"
	"os"
	
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// AddDevice creates a new device for the user
func (rest *Rest) AddDevice(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	
	type AddDeviceRequest struct {
		Name string `json:"name" validate:"required"`
	}
	
	var req AddDeviceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error": "Invalid request body",
		})
	}
	
	// Create new device in database
	userRepo := repository.GetUserRepository()
	device := &models.UserDevice{
		ID:         uuid.New().String(),
		UserID:     userID,
		DeviceName: req.Name,
		Status:     "offline",
	}
	
	if err := userRepo.CreateDevice(device); err != nil {
		logrus.Errorf("Failed to create device: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error": "Failed to create device",
		})
	}
	
	logrus.Infof("Created new device %s for user %s", device.ID, userID)
	
	return c.JSON(fiber.Map{
		"success": true,
		"device": device,
	})
}

// ClearDeviceData clears WhatsApp connection data for a device
func (rest *Rest) ClearDeviceData(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	userID := c.Locals("userID").(string)
	
	// Verify device belongs to user
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDevice(userID, deviceID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"success": false,
			"error": "Device not found",
		})
	}
	
	// Disconnect WhatsApp client if connected
	cm := whatsapp.GetClientManager()
	if client := cm.GetClient(deviceID); client != nil {
		client.Disconnect()
		cm.RemoveClient(deviceID)
		logrus.Infof("Disconnected WhatsApp client for device %s", deviceID)
	}
	
	// Clear WhatsApp session data from database
	if rest.waDB != nil {
		// Get all devices from WhatsApp store
		devices, err := rest.waDB.GetAllDevices()
		if err == nil {
			for _, waDevice := range devices {
				// Check if this WhatsApp device belongs to our device ID
				// We'll need to match by JID if stored, or clear orphaned devices
				if device.JID != "" && waDevice.ID != nil && waDevice.ID.String() == device.JID {
					err := waDevice.Delete()
					if err != nil {
						logrus.Errorf("Failed to delete WhatsApp device data: %v", err)
					} else {
						logrus.Infof("Deleted WhatsApp device data for %s", device.JID)
					}
				}
			}
		}
	}
	
	// Update device status
	err = userRepo.UpdateDeviceStatus(deviceID, "offline", "", "")
	if err != nil {
		logrus.Errorf("Failed to update device status: %v", err)
	}
	
	// Clear QR code files if any
	qrPattern := filepath.Join(config.PathQrCode, fmt.Sprintf("*%s*.png", deviceID))
	matches, _ := filepath.Glob(qrPattern)
	for _, match := range matches {
		os.Remove(match)
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Device data cleared successfully",
	})
}

// GetDeviceQRCode generates a new QR code for device connection
func (rest *Rest) GetDeviceQRCode(c *fiber.Ctx) error {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "device_id is required",
		})
	}
	
	userID := c.Locals("userID").(string)
	
	// Verify device belongs to user
	userRepo := repository.GetUserRepository()
	device, err := userRepo.GetDevice(userID, deviceID)
	if err != nil {
		return c.Status(403).JSON(utils.ResponseData{
			Status:  403,
			Code:    "FORBIDDEN",
			Message: "Device not found or access denied",
		})
	}
	
	// Store connection session
	whatsapp.StoreConnectionSession(userID, &whatsapp.ConnectionSession{
		UserID:   userID,
		DeviceID: deviceID,
	})
	
	// Check if we already have a client for this device
	cm := whatsapp.GetClientManager()
	existingClient := cm.GetClient(deviceID)
	
	if existingClient != nil && existingClient.IsConnected() && existingClient.IsLoggedIn() {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "ALREADY_CONNECTED",
			Message: "Device is already connected",
		})
	}
	
	// Create a new WhatsApp device specifically for this device ID
	waDevice := rest.waDB.NewDevice()
	
	// Create new WhatsApp client for this device
	client := whatsapp.CreateNewClient(waDevice)
	
	// Connect the client
	err = client.Connect()
	if err != nil {
		logrus.Errorf("Failed to connect WhatsApp client: %v", err)
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "CONNECTION_ERROR",
			Message: "Failed to connect to WhatsApp",
		})
	}
	
	// Get QR channel
	qrChan, err := client.GetQRChannel(c.Context())
	if err != nil {
		// If already logged in, update device status
		if client.IsLoggedIn() && client.Store.ID != nil {
			userRepo.UpdateDeviceStatus(deviceID, "online", client.Store.ID.User, client.Store.ID.String())
			cm.AddClient(deviceID, client)
			
			return c.JSON(utils.ResponseData{
				Status:  200,
				Code:    "ALREADY_LOGGED_IN",
				Message: "Device already logged in",
				Results: map[string]interface{}{
					"phone": client.Store.ID.User,
					"jid":   client.Store.ID.String(),
				},
			})
		}
		
		logrus.Errorf("Failed to get QR channel: %v", err)
		client.Disconnect()
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "QR_ERROR",
			Message: "Failed to generate QR code",
		})
	}
	
	// Process QR codes in background
	go rest.processDeviceQRCodes(deviceID, device.DeviceName, client, qrChan)
	
	// Return the QR code endpoint
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "QR code generation started",
		Results: map[string]interface{}{
			"qr_endpoint": fmt.Sprintf("/app/device-qr/%s", deviceID),
			"device_id":   deviceID,
			"device_name": device.DeviceName,
		},
	})
}

// processDeviceQRCodes handles QR codes for a specific device
func (rest *Rest) processDeviceQRCodes(deviceID, deviceName string, client *whatsmeow.Client, qrChan <-chan whatsmeow.QRChannelItem) {
	for evt := range qrChan {
		if evt.Event == "code" {
			logrus.Infof("QR Code received for device %s", deviceID)
			
			// Save QR code image
			qrPath := filepath.Join(config.PathQrCode, fmt.Sprintf("device-%s.png", deviceID))
			err := utils.GenerateQRCode(evt.Code, qrPath)
			if err != nil {
				logrus.Errorf("Failed to generate QR code image: %v", err)
				continue
			}
			
			// Update device with QR info
			whatsapp.StoreDeviceQR(deviceID, qrPath, evt.Code)
		}
	}
	
	// After QR channel closes, check if logged in
	if client.IsLoggedIn() && client.Store.ID != nil {
		logrus.Infof("Device %s successfully logged in as %s", deviceID, client.Store.ID.String())
		
		// Update device status
		userRepo := repository.GetUserRepository()
		userRepo.UpdateDeviceStatus(deviceID, "online", client.Store.ID.User, client.Store.ID.String())
		
		// Register client with manager
		cm := whatsapp.GetClientManager()
		cm.AddClient(deviceID, client)
		
		// Clear connection session
		sessions := whatsapp.GetAllConnectionSessions()
		for userID, session := range sessions {
			if session != nil && session.DeviceID == deviceID {
				whatsapp.ClearConnectionSession(userID)
				break
			}
		}
	}
}

// ServeDeviceQR serves the QR code image for a specific device
func (rest *Rest) ServeDeviceQR(c *fiber.Ctx) error {
	deviceID := c.Params("deviceId")
	
	// Get QR info
	qrInfo := whatsapp.GetDeviceQR(deviceID)
	if qrInfo == nil || qrInfo.ImagePath == "" {
		// Return a placeholder image or error
		return c.Status(404).JSON(fiber.Map{
			"error": "QR code not found",
		})
	}
	
	// Check if file exists
	if _, err := os.Stat(qrInfo.ImagePath); os.IsNotExist(err) {
		return c.Status(404).JSON(fiber.Map{
			"error": "QR code file not found",
		})
	}
	
	// Serve the QR code image
	return c.SendFile(qrInfo.ImagePath)
}
