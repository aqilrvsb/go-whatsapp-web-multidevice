package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"golang.org/x/net/proxy"
)

// CreateProxiedClient creates a WhatsApp client with proxy support
func CreateProxiedClient(device *sqlstore.Device, deviceID string) (*whatsmeow.Client, error) {
	// Get proxy manager
	pm := GetProxyManager()
	
	// Assign a proxy to this device
	assignedProxy, err := pm.AssignProxyToDevice(deviceID)
	if err != nil {
		// If no proxy available, create normal client
		return whatsmeow.NewClient(device, nil), nil
	}
	
	// Create proxied HTTP client
	httpClient, err := createProxiedHTTPClient(assignedProxy)
	if err != nil {
		pm.ReleaseProxy(deviceID)
		return nil, fmt.Errorf("failed to create proxied client: %v", err)
	}
	
	// Create WhatsApp client with custom HTTP client
	client := whatsmeow.NewClient(device, nil)
	
	// Override the HTTP client
	client.HTTPClient = httpClient
	
	return client, nil
}

// createProxiedHTTPClient creates an HTTP client with proxy
func createProxiedHTTPClient(p *Proxy) (*http.Client, error) {
	switch p.Type {
	case "http", "https":
		proxyURL := fmt.Sprintf("http://%s:%s", p.Host, p.Port)
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxy),
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		}, nil
		
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", p.Host, p.Port), nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		
		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
		}, nil
		
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", p.Type)
	}
}