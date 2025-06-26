package whatsapp

import (
	"context"
	"fmt"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/proxy"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// InitWaCLIWithProxy initializes WhatsApp client with proxy support
func InitWaCLIWithProxy(ctx context.Context, storeContainer *sqlstore.Container, deviceID string) *whatsmeow.Client {
	device, err := storeContainer.GetFirstDevice(ctx)
	if err != nil {
		log.Errorf("Failed to get device: %v", err)
		panic(err)
	}

	if device == nil {
		log.Errorf("No device found")
		panic("No device found")
	}

	// Configure device properties
	osName := fmt.Sprintf("%s %s", config.AppOs, config.AppVersion)
	store.DeviceProps.PlatformType = &config.AppPlatform
	store.DeviceProps.Os = &osName

	var cli *whatsmeow.Client

	// Check if proxy is enabled
	if config.ProxyEnabled {
		// Create client with proxy
		cli, err = proxy.CreateProxiedClient(device, deviceID)
		if err != nil {
			log.Warnf("Failed to create proxied client: %v, falling back to normal client", err)
			cli = whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
		} else {
			log.Infof("Created proxied client for device %s", deviceID)
		}
	} else {
		// Create normal client
		cli = whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
	}

	// Configure client
	cli.EnableAutoReconnect = true
	cli.AutoTrustIdentity = true
	cli.AddEventHandler(func(rawEvt interface{}) {
		handler(ctx, rawEvt)
	})

	// Set as global client for debugging
	SetGlobalClient(cli)

	return cli
}