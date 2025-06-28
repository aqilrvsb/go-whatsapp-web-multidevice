package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/validations"
	fiberUtils "github.com/gofiber/fiber/v2/utils"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type serviceApp struct {
	WaCli *whatsmeow.Client
	db    *sqlstore.Container
}

func NewAppService(waCli *whatsmeow.Client, db *sqlstore.Container) domainApp.IAppUsecase {
	return &serviceApp{
		WaCli: waCli,
		db:    db,
	}
}

func (service serviceApp) Login(ctx context.Context) (response domainApp.LoginResponse, err error) {
	// CRITICAL FIX: Always use a fresh database container for multi-device support
	if service.db == nil {
		return response, fmt.Errorf("database not initialized")
	}
	
	// FIX 1: Create a new device store for each login attempt
	logrus.Info("Creating new WhatsApp device for login...")
	
	// Get a fresh device from the store
	device := service.db.NewDevice()
	
	// Create a fresh WhatsApp client for this device
	newClient := whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
	
	// IMPORTANT: Don't override the service client, use local client
	// This allows multiple devices to connect simultaneously
	
	// FIX 2: Add event handler to properly register device after successful login
	newClient.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			logrus.Infof("Pair success event: %s", v.ID.String())
			// Device is paired but not fully connected yet
		case *events.Connected:
			logrus.Info("Connected event received - device fully connected!")
			// Now register the device with ClientManager
			service.registerDeviceAfterConnection(newClient)
		case *events.LoggedOut:
			logrus.Warn("Device logged out")
		}
	})
	
	// Connect the new client
	logrus.Info("Connecting new WhatsApp client...")
	err = newClient.Connect()
	if err != nil {
		logrus.Error("Error when connect to whatsapp: ", err)
		return response, pkgError.ErrReconnect
	}

	// Check if this device is already logged in
	if newClient.IsLoggedIn() {
		logrus.Info("Device already logged in")
		// Register it immediately
		service.registerDeviceAfterConnection(newClient)
		return response, pkgError.ErrAlreadyLoggedIn
	}

	// Get QR channel
	logrus.Info("Getting QR channel...")
	ch, err := newClient.GetQRChannel(context.Background())
	if err != nil {
		logrus.Error("Error getting QR channel: ", err.Error())
		
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			// This device is already registered, try to delete and recreate
			logrus.Warn("Device already has ID, deleting and recreating...")
			device.Delete(ctx)
			
			// Create a completely new device
			device = service.db.NewDevice()
			newClient = whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
			
			// Try connecting again
			err = newClient.Connect()
			if err != nil {
				return response, fmt.Errorf("failed to reconnect after device reset: %w", err)
			}
			
			// Try getting QR channel again
			ch, err = newClient.GetQRChannel(context.Background())
			if err != nil {
				return response, fmt.Errorf("failed to get QR channel after reset: %w", err)
			}
		} else {
			return response, pkgError.ErrQrChannel
		}
	}

	logrus.Info("Waiting for QR code...")
	
	// Process QR codes in a separate goroutine
	go func() {
		for evt := range ch {
			if evt.Event == "code" {
				logrus.Infof("QR code update: timeout=%v", evt.Timeout/time.Second)
				
				// Only process the first QR code for the response
				if response.Code == "" {
					response.Code = evt.Code
					response.Duration = evt.Timeout / time.Second / 2
					
					// Generate QR image
					qrDir := config.PathQrCode
					if err := os.MkdirAll(qrDir, 0755); err != nil {
						logrus.Errorf("Failed to create QR directory: %v", err)
						continue
					}
					
					qrPath := fmt.Sprintf("%s/scan-qr-%s.png", qrDir, fiberUtils.UUIDv4())
					err := qrcode.WriteFile(evt.Code, qrcode.Medium, 512, qrPath)
					if err != nil {
						logrus.Error("Error writing QR code: ", err)
						continue
					}
					
					response.ImagePath = qrPath
					
					// Cleanup after timeout
					go func(path string, timeout time.Duration) {
						time.Sleep(timeout)
						os.Remove(path)
					}(qrPath, response.Duration*time.Second)
				}
			}
		}
		logrus.Info("QR channel closed")
	}()
	
	// Wait for first QR code with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			return response, fmt.Errorf("timeout waiting for QR code")
		case <-ticker.C:
			if response.Code != "" {
				// Start monitoring for successful login
				go service.monitorLoginForClient(newClient)
				return response, nil
			}
		}
	}
}

// FIX: Separate monitor function for each client
func (service serviceApp) monitorLoginForClient(client *whatsmeow.Client) {
	logrus.Info("Starting login monitor for new device...")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for i := 0; i < 30; i++ { // 60 second timeout
		<-ticker.C
		if client.IsLoggedIn() {
			logrus.Info("Login successful! Device connected.")
			// Registration will be handled by the Connected event
			return
		}
		if i%5 == 0 {
			logrus.Infof("Still waiting for login... (%d seconds)", i*2)
		}
	}
	logrus.Warn("Login monitor timeout after 60 seconds")
}

// FIX: Proper device registration after successful connection
func (service serviceApp) registerDeviceAfterConnection(client *whatsmeow.Client) {
	if client.Store.ID == nil {
		logrus.Warn("Cannot register device - no JID available")
		return
	}
	
	jid := client.Store.ID.String()
	phoneNumber := client.Store.ID.User
	
	logrus.Infof("Registering device - JID: %s, Phone: %s", jid, phoneNumber)
	
	// Check all connection sessions to find the matching device
	allSessions := whatsapp.GetAllConnectionSessions()
	
	for userID, session := range allSessions {
		if session != nil && session.DeviceID != "" {
			logrus.Infof("Found session for user %s, device %s", userID, session.DeviceID)
			
			// Register the client with ClientManager
			cm := whatsapp.GetClientManager()
			cm.AddClient(session.DeviceID, client)
			logrus.Infof("Successfully registered device %s with ClientManager", session.DeviceID)
			
			// Update device status in database
			// This is handled in the Connected event handler in init.go
			
			// Clear the session
			whatsapp.ClearConnectionSession(userID)
			break
		}
	}
}

// Keep the old monitor function for backward compatibility
func (service serviceApp) monitorLogin() {
	service.monitorLoginForClient(service.WaCli)
}

func (service serviceApp) LoginWithCode(ctx context.Context, phoneNumber string) (loginCode string, err error) {
	if err = validations.ValidateLoginWithCode(ctx, phoneNumber); err != nil {
		logrus.Errorf("Error when validate login with code: %s", err.Error())
		return loginCode, err
	}

	// For multi-device, create a new client
	device := service.db.NewDevice()
	newClient := whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
	
	// Connect the client
	err = newClient.Connect()
	if err != nil {
		return loginCode, fmt.Errorf("failed to connect: %w", err)
	}
	
	// Check if already logged in
	if newClient.IsLoggedIn() {
		logrus.Warn("Device is already logged in")
		return loginCode, pkgError.ErrAlreadyLoggedIn
	}

	loginCode, err = newClient.PairPhone(ctx, phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		logrus.Errorf("Error when pairing with phone: %s", err.Error())
		return loginCode, err
	}
	
	// Add event handler for this client too
	newClient.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Connected:
			service.registerDeviceAfterConnection(newClient)
		}
	})

	return loginCode, nil
}

func (service serviceApp) Logout(ctx context.Context) (err error) {
	if service.WaCli == nil || !service.WaCli.IsConnected() {
		return pkgError.ErrNotConnected
	}

	err = service.WaCli.Logout(ctx)
	if err != nil {
		return err
	}

	service.WaCli.Disconnect()
	return nil
}

func (service serviceApp) Reconnect(ctx context.Context) (err error) {
	if service.WaCli == nil {
		return fmt.Errorf("whatsapp client is not initialized")
	}
	
	service.WaCli.Disconnect()
	
	// Wait a bit before reconnecting
	time.Sleep(2 * time.Second)
	
	return service.WaCli.Connect()
}

func (service serviceApp) FirstDevice(ctx context.Context) (response domainApp.DevicesResponse, err error) {
	if service.db == nil {
		return response, fmt.Errorf("database not initialized")
	}
	
	device, err := service.db.GetFirstDevice(ctx)
	if err != nil {
		return response, err
	}
	
	if device == nil {
		return response, fmt.Errorf("no device found")
	}
	
	response = domainApp.DevicesResponse{
		Name:   device.PushName,
		Device: device.ID.String(),
	}
	
	return response, nil
}

func (service serviceApp) FetchDevices(ctx context.Context) (response []domainApp.DevicesResponse, err error) {
	if service.db == nil {
		return response, fmt.Errorf("database not initialized")
	}
	
	devices, err := service.db.GetAllDevices(ctx)
	if err != nil {
		return response, err
	}
	
	response = make([]domainApp.DevicesResponse, 0, len(devices))
	for _, device := range devices {
		response = append(response, domainApp.DevicesResponse{
			Name:   device.PushName,
			Device: device.ID.String(),
		})
	}
	
	return response, nil
}