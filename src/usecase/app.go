package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainApp "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/app"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/validations"
	fiberUtils "github.com/gofiber/fiber/v2/utils"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
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

func (service serviceApp) Login(_ context.Context) (response domainApp.LoginResponse, err error) {
	if service.WaCli == nil {
		return response, pkgError.ErrWaCLI
	}

	// Disconnect for reconnecting
	service.WaCli.Disconnect()

	chImage := make(chan string)
	var loginComplete chan bool

	ch, err := service.WaCli.GetQRChannel(context.Background())
	if err != nil {
		logrus.Error(err.Error())
		// This error means that we're already logged in, so ignore it.
		if errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			_ = service.WaCli.Connect() // just connect to websocket
			if service.WaCli.IsLoggedIn() {
				return response, pkgError.ErrAlreadyLoggedIn
			}
			return response, pkgError.ErrSessionSaved
		} else {
			return response, pkgError.ErrQrChannel
		}
	} else {
		loginComplete := make(chan bool, 1)
		
		go func() {
			for evt := range ch {
				logrus.Infof("QR Code received, timeout: %v seconds", evt.Timeout/time.Second)
				response.Code = evt.Code
				response.Duration = evt.Timeout / time.Second / 2
				if evt.Code != "" {
					qrPath := fmt.Sprintf("%s/scan-qr-%s.png", config.PathQrCode, fiberUtils.UUIDv4())
					err = qrcode.WriteFile(evt.Code, qrcode.Medium, 512, qrPath)
					if err != nil {
						logrus.Error("Error when write qr code to file: ", err)
					}
					go func() {
						time.Sleep(response.Duration * time.Second)
						err := os.Remove(qrPath)
						if err != nil {
							logrus.Error("error when remove qrImage file", err.Error())
						}
					}()
					chImage <- qrPath
				}
			}
			logrus.Info("QR channel closed - login might be complete")
			loginComplete <- true
		}()
	}

	err = service.WaCli.Connect()
	if err != nil {
		logger.Error("Error when connect to whatsapp", err)
		return response, pkgError.ErrReconnect
	}
	
	response.ImagePath = <-chImage
	
	// Wait for login completion with better monitoring
	go func() {
		logrus.Info("Starting login monitor...")
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for i := 0; i < 60; i++ { // 60 second timeout
			<-ticker.C
			if service.WaCli.IsLoggedIn() {
				logrus.Info("Login detected! Registering device...")
				
				// Get device info from current session
				allSessions := whatsapp.GetAllConnectionSessions()
				for userID, session := range allSessions {
					if session != nil && session.DeviceID != "" {
						logrus.Infof("Registering device %s for user %s", session.DeviceID, userID)
						cm := whatsapp.GetClientManager()
						cm.AddClient(session.DeviceID, service.WaCli)
						break
					}
				}
				
				loginComplete <- true
				return
			}
			if i%5 == 0 {
				logrus.Infof("Still waiting for login... (%d seconds)", i)
			}
		}
		logrus.Warn("Login monitor timeout")
		loginComplete <- false
	}()
	
	// Wait for completion
	<-loginComplete
	logrus.Info("Login process completed")

	return response, nil
}

func (service serviceApp) LoginWithCode(ctx context.Context, phoneNumber string) (loginCode string, err error) {
	if err = validations.ValidateLoginWithCode(ctx, phoneNumber); err != nil {
		logrus.Errorf("Error when validate login with code: %s", err.Error())
		return loginCode, err
	}

	// detect is already logged in
	if service.WaCli.Store.ID != nil {
		logrus.Warn("User is already logged in")
		return loginCode, pkgError.ErrAlreadyLoggedIn
	}

	// reconnect first
	_ = service.Reconnect(ctx)

	loginCode, err = service.WaCli.PairPhone(ctx, phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		logrus.Errorf("Error when pairing phone: %s", err.Error())
		return loginCode, err
	}

	logrus.Infof("Successfully paired phone with code: %s", loginCode)
	return loginCode, nil
}

func (service serviceApp) Logout(ctx context.Context) (err error) {
	// delete history
	files, err := filepath.Glob(fmt.Sprintf("./%s/history-*", config.PathStorages))
	if err != nil {
		return err
	}

	for _, f := range files {
		err = os.Remove(f)
		if err != nil {
			return err
		}
	}
	// delete qr images
	qrImages, err := filepath.Glob(fmt.Sprintf("./%s/scan-*", config.PathQrCode))
	if err != nil {
		return err
	}

	for _, f := range qrImages {
		err = os.Remove(f)
		if err != nil {
			return err
		}
	}

	// delete senditems
	qrItems, err := filepath.Glob(fmt.Sprintf("./%s/*", config.PathSendItems))
	if err != nil {
		return err
	}

	for _, f := range qrItems {
		if !strings.Contains(f, ".gitignore") {
			err = os.Remove(f)
			if err != nil {
				return err
			}
		}
	}

	err = service.WaCli.Logout(ctx)
	return
}

func (service serviceApp) Reconnect(_ context.Context) (err error) {
	service.WaCli.Disconnect()
	return service.WaCli.Connect()
}

func (service serviceApp) FirstDevice(ctx context.Context) (response domainApp.DevicesResponse, err error) {
	if service.WaCli == nil {
		return response, pkgError.ErrWaCLI
	}

	devices, err := service.db.GetFirstDevice(ctx)
	if err != nil {
		return response, err
	}

	response.Device = devices.ID.String()
	if devices.PushName != "" {
		response.Name = devices.PushName
	} else {
		response.Name = devices.BusinessName
	}

	return response, nil
}

func (service serviceApp) FetchDevices(ctx context.Context) (response []domainApp.DevicesResponse, err error) {
	if service.WaCli == nil {
		return response, pkgError.ErrWaCLI
	}

	devices, err := service.db.GetAllDevices(ctx)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		var d domainApp.DevicesResponse
		d.Device = device.ID.String()
		if device.PushName != "" {
			d.Name = device.PushName
		} else {
			d.Name = device.BusinessName
		}

		response = append(response, d)
	}

	return response, nil
}
