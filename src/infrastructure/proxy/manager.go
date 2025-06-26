package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

// ProxyManager manages proxy allocation for devices
type ProxyManager struct {
	proxies      []Proxy
	deviceProxies map[string]*Proxy  // deviceID -> proxy mapping
	mu           sync.RWMutex
	updateTicker *time.Ticker
}

// Proxy represents a proxy server
type Proxy struct {
	Type     string    `json:"type"`     // http, socks5
	Host     string    `json:"host"`
	Port     string    `json:"port"`
	Country  string    `json:"country"`
	Speed    float64   `json:"speed"`    // ms
	LastCheck time.Time `json:"last_check"`
	Working  bool      `json:"working"`
	InUse    bool      `json:"in_use"`
}

var (
	manager     *ProxyManager
	managerOnce sync.Once
)

// GetProxyManager returns singleton proxy manager
func GetProxyManager() *ProxyManager {
	managerOnce.Do(func() {
		manager = &ProxyManager{
			proxies:       make([]Proxy, 0),
			deviceProxies: make(map[string]*Proxy),
		}
		
		// Start proxy updater
		manager.StartAutoUpdate()
		
		// Initial fetch
		go manager.FetchMalaysianProxies()
	})
	return manager
}

// StartAutoUpdate starts automatic proxy list updates
func (pm *ProxyManager) StartAutoUpdate() {
	pm.updateTicker = time.NewTicker(30 * time.Minute)
	
	go func() {
		for range pm.updateTicker.C {
			pm.FetchMalaysianProxies()
		}
	}()
}

// FetchMalaysianProxies fetches free Malaysian proxies from multiple sources
func (pm *ProxyManager) FetchMalaysianProxies() {
	logrus.Info("Fetching Malaysian proxies...")
	
	var allProxies []Proxy
	
	// Source 1: Free proxy APIs
	proxies1 := pm.fetchFromProxyList()
	allProxies = append(allProxies, proxies1...)
	
	// Source 2: GitHub proxy lists
	proxies2 := pm.fetchFromGitHub()
	allProxies = append(allProxies, proxies2...)
	
	// Source 3: ProxyScrape API
	proxies3 := pm.fetchFromProxyScrape()
	allProxies = append(allProxies, proxies3...)
	
	// Filter and validate Malaysian proxies
	malaysianProxies := pm.filterMalaysianProxies(allProxies)
	
	// Test proxies
	workingProxies := pm.testProxies(malaysianProxies)
	
	pm.mu.Lock()
	pm.proxies = workingProxies
	pm.mu.Unlock()
	
	logrus.Infof("Found %d working Malaysian proxies", len(workingProxies))
}

// fetchFromProxyList fetches from proxy-list.download
func (pm *ProxyManager) fetchFromProxyList() []Proxy {
	var proxies []Proxy
	
	// API endpoint for Malaysian proxies
	urls := []string{
		"https://www.proxy-list.download/api/v1/get?type=http&country=MY",
		"https://www.proxy-list.download/api/v1/get?type=socks5&country=MY",
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	
	for _, apiURL := range urls {
		resp, err := client.Get(apiURL)
		if err != nil {
			logrus.Warnf("Failed to fetch from proxy-list: %v", err)
			continue
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		
		lines := strings.Split(string(body), "\n")
		proxyType := "http"
		if strings.Contains(apiURL, "socks5") {
			proxyType = "socks5"
		}
		
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				proxies = append(proxies, Proxy{
					Type:    proxyType,
					Host:    parts[0],
					Port:    parts[1],
					Country: "MY",
				})
			}
		}
	}
	
	return proxies
}// fetchFromGitHub fetches from GitHub proxy lists
func (pm *ProxyManager) fetchFromGitHub() []Proxy {
	var proxies []Proxy
	
	// Popular GitHub proxy lists
	urls := []string{
		"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/http.txt",
		"https://raw.githubusercontent.com/TheSpeedX/PROXY-List/master/socks5.txt",
		"https://raw.githubusercontent.com/clarketm/proxy-list/master/proxy-list-raw.txt",
		"https://raw.githubusercontent.com/sunny9577/proxy-scraper/master/proxies.txt",
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	
	for _, apiURL := range urls {
		resp, err := client.Get(apiURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				proxies = append(proxies, Proxy{
					Type: "http",
					Host: parts[0],
					Port: parts[1],
				})
			}
		}
	}
	
	return proxies
}
// fetchFromProxyScrape fetches from ProxyScrape API
func (pm *ProxyManager) fetchFromProxyScrape() []Proxy {
	var proxies []Proxy
	
	// ProxyScrape API endpoints
	urls := []string{
		"https://api.proxyscrape.com/v2/?request=get&protocol=http&timeout=10000&country=MY&ssl=all&anonymity=all&format=textplain",
		"https://api.proxyscrape.com/v2/?request=get&protocol=socks5&timeout=10000&country=MY&format=textplain",
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	
	for _, apiURL := range urls {
		resp, err := client.Get(apiURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}
		
		proxyType := "http"
		if strings.Contains(apiURL, "socks5") {
			proxyType = "socks5"
		}
		
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				proxies = append(proxies, Proxy{
					Type:    proxyType,
					Host:    parts[0],
					Port:    parts[1],
					Country: "MY",
				})
			}
		}
	}
	
	return proxies
}
// filterMalaysianProxies filters proxies by Malaysian IP ranges
func (pm *ProxyManager) filterMalaysianProxies(proxies []Proxy) []Proxy {
	var malaysianProxies []Proxy
	
	// Malaysian IP ranges (simplified - you can add more)
	malaysianIPRanges := []string{
		"103.6.", "103.8.", "103.16.", "103.18.", "103.26.",
		"103.30.", "103.52.", "103.86.", "103.94.", "103.106.",
		"103.107.", "103.233.", "175.136.", "175.137.", "175.138.",
		"175.139.", "175.140.", "175.141.", "175.142.", "175.143.",
		"175.144.", "202.71.", "202.75.", "203.82.", "203.106.",
		"210.48.", "210.186.", "218.208.", "219.92.", "219.93.",
	}
	
	for _, proxy := range proxies {
		// Check if proxy is from Malaysia
		if proxy.Country == "MY" {
			malaysianProxies = append(malaysianProxies, proxy)
			continue
		}
		
		// Check IP ranges
		for _, prefix := range malaysianIPRanges {
			if strings.HasPrefix(proxy.Host, prefix) {
				proxy.Country = "MY"
				malaysianProxies = append(malaysianProxies, proxy)
				break
			}
		}
	}
	
	return malaysianProxies
}
// testProxies tests proxy connectivity
func (pm *ProxyManager) testProxies(proxies []Proxy) []Proxy {
	var workingProxies []Proxy
	var wg sync.WaitGroup
	resultChan := make(chan Proxy, len(proxies))
	
	// Limit concurrent tests
	semaphore := make(chan struct{}, 50)
	
	for _, proxy := range proxies {
		wg.Add(1)
		go func(p Proxy) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			if pm.testProxy(p) {
				p.Working = true
				p.LastCheck = time.Now()
				resultChan <- p
			}
		}(proxy)
	}
	
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	for proxy := range resultChan {
		workingProxies = append(workingProxies, proxy)
	}
	
	return workingProxies
}
// testProxy tests a single proxy
func (pm *ProxyManager) testProxy(p Proxy) bool {
	timeout := 5 * time.Second
	
	// Test URL - WhatsApp web endpoint
	testURL := "https://web.whatsapp.com"
	
	var client *http.Client
	
	switch p.Type {
	case "http", "https":
		proxyURL := fmt.Sprintf("http://%s:%s", p.Host, p.Port)
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return false
		}
		
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxy),
				DialContext: (&net.Dialer{
					Timeout: timeout,
				}).DialContext,
			},
			Timeout: timeout,
		}
		
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%s", p.Host, p.Port), nil, proxy.Direct)
		if err != nil {
			return false
		}
		
		client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
			Timeout: timeout,
		}
		
	default:
		return false
	}
	
	start := time.Now()
	resp, err := client.Get(testURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Calculate speed
	p.Speed = float64(time.Since(start).Milliseconds())
	
	return resp.StatusCode == 200
}
// AssignProxyToDevice assigns a proxy to a device
func (pm *ProxyManager) AssignProxyToDevice(deviceID string) (*Proxy, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Check if device already has a proxy
	if proxy, exists := pm.deviceProxies[deviceID]; exists && proxy.Working {
		return proxy, nil
	}
	
	// Find an available proxy
	for i := range pm.proxies {
		if !pm.proxies[i].InUse && pm.proxies[i].Working {
			pm.proxies[i].InUse = true
			pm.deviceProxies[deviceID] = &pm.proxies[i]
			
			logrus.Infof("Assigned proxy %s:%s to device %s", 
				pm.proxies[i].Host, pm.proxies[i].Port, deviceID)
			
			return &pm.proxies[i], nil
		}
	}
	
	return nil, fmt.Errorf("no available proxies")
}

// GetProxyForDevice gets the assigned proxy for a device
func (pm *ProxyManager) GetProxyForDevice(deviceID string) *Proxy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return pm.deviceProxies[deviceID]
}

// ReleaseProxy releases a proxy from a device
func (pm *ProxyManager) ReleaseProxy(deviceID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if proxy, exists := pm.deviceProxies[deviceID]; exists {
		proxy.InUse = false
		delete(pm.deviceProxies, deviceID)
		logrus.Infof("Released proxy from device %s", deviceID)
	}
}

// GetAvailableProxyCount returns the number of available proxies
func (pm *ProxyManager) GetAvailableProxyCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	count := 0
	for _, proxy := range pm.proxies {
		if !proxy.InUse && proxy.Working {
			count++
		}
	}
	
	return count
}

// GetProxyStats returns proxy statistics
func (pm *ProxyManager) GetProxyStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total":     len(pm.proxies),
		"working":   0,
		"in_use":    0,
		"available": 0,
		"by_type":   make(map[string]int),
	}
	
	for _, proxy := range pm.proxies {
		if proxy.Working {
			stats["working"] = stats["working"].(int) + 1
		}
		if proxy.InUse {
			stats["in_use"] = stats["in_use"].(int) + 1
		}
		if proxy.Working && !proxy.InUse {
			stats["available"] = stats["available"].(int) + 1
		}
		
		typeStats := stats["by_type"].(map[string]int)
		typeStats[proxy.Type]++
	}
	
	return stats
}