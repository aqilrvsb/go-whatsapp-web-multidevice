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
	// For multi-device support, we need to create a new client for this login attempt
	if service.db == nil {
		return response, fmt.Errorf("database not initialized")
	}
	
	// Create a new device in the store
	logrus.Info("Creating new WhatsApp device for login...")
	device := service.db.NewDevice()
	
	// Create a new WhatsApp client for this device
	newClient := whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
	
	// Add event handler to properly register device after successful login
	newClient.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			logrus.Infof("Pair success event: %s", v.ID.String())
		case *events.Connected:
			logrus.Info("Connected event received - device fully connected!")
			service.registerDeviceAfterConnection(newClient)
			// Keep the client alive by adding keepalive monitoring
			go func(client *whatsmeow.Client) {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				
				for range ticker.C {
					if !client.IsConnected() {
						logrus.Warn("Client disconnected, attempting reconnect...")
						client.Connect()
					}
				}
			}(newClient)
		case *events.LoggedOut:
			logrus.Warn("Device logged out")
		}
	})
	
	// IMPORTANT: Get QR channel BEFORE connecting (like the working version)
	logrus.Info("Getting QR channel...")
	ch, err := newClient.GetQRChannel(ctx)
	if err != nil {
		logrus.Error("Error getting QR channel: ", err.Error())
		
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			// Already logged in, just connect
			err = newClient.Connect()
			if err != nil {
				return response, fmt.Errorf("failed to connect: %w", err)
			}
			if newClient.IsLoggedIn() {
				service.registerDeviceAfterConnection(newClient)
				return response, pkgError.ErrAlreadyLoggedIn
			}
			return response, pkgError.ErrSessionSaved
		} else {
			return response, pkgError.ErrQrChannel
		}
	}
	
	// Setup QR processing like the working version
	chImage := make(chan string)
	stopQR := make(chan bool, 1)
	
	go func() {
		for {
			select {
			case evt := <-ch:
				response.Code = evt.Code
				response.Duration = evt.Timeout / time.Second / 2
				if evt.Event == "code" {
					qrPath := fmt.Sprintf("%s/scan-qr-%s.png", config.PathQrCode, fiberUtils.UUIDv4())
					err := qrcode.WriteFile(evt.Code, qrcode.Medium, 512, qrPath)
					if err != nil {
						logrus.Error("Error when write qr code to file: ", err)
						continue
					}
					
					// Cleanup after timeout
					go func() {
						time.Sleep(response.Duration * time.Second)
						os.Remove(qrPath)
					}()
					
					// Only send first QR image
					select {
					case chImage <- qrPath:
					default:
					}
				} else {
					logrus.Infof("QR event: %s", evt.Event)
				}
			case <-stopQR:
				logrus.Info("Stopping QR generation - device connected")
				return
			}
		}
	}()
	
	// NOW connect AFTER setting up QR channel (like the working version)
	logrus.Info("Connecting WhatsApp client...")
	err = newClient.Connect()
	if err != nil {
		logrus.Error("Error when connect to whatsapp: ", err)
		return response, pkgError.ErrReconnect
	}
	
	// Wait for QR image path
	select {
	case imagePath := <-chImage:
		response.ImagePath = imagePath
		logrus.Infof("QR code generated: %s", imagePath)
	case <-time.After(60 * time.Second):
		close(stopQR)
		return response, fmt.Errorf("timeout waiting for QR code")
	}
	
	// Monitor for successful connection in background
	go func() {
		// Check periodically if device is connected
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		timeout := time.After(5 * time.Minute)
		
		for {
			select {
			case <-ticker.C:
				if newClient.IsLoggedIn() {
					logrus.Info("Device successfully logged in!")
					close(stopQR) // Stop QR generation
					return
				}
			case <-timeout:
				logrus.Warn("Connection monitoring timeout")
				close(stopQR)
				return
			}
		}
	}()
	
	return response, nil
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
			
			// IMPORTANT: Keep a reference to prevent garbage collection
			// The ClientManager should maintain this reference
			
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