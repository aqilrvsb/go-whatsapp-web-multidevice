package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/antipattern"
	"github.com/sirupsen/logrus"
)

// PlatformSender handles sending messages via external platforms
type PlatformSender struct {
	client            *http.Client
	messageRandomizer *antipattern.MessageRandomizer
	greetingProcessor *antipattern.GreetingProcessor
}

// NewPlatformSender creates a new platform sender
func NewPlatformSender() *PlatformSender {
	return &PlatformSender{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		messageRandomizer: antipattern.NewMessageRandomizer(),
		greetingProcessor: antipattern.NewGreetingProcessor(),
	}
}

// SendMessage sends a message via external platform
// NOTE: Anti-spam is already applied by BroadcastWorker - we just send raw content
func (ps *PlatformSender) SendMessage(platform, instance, phone, recipientName, message, imageURL, deviceID string) error {
	// Log initial request
	logrus.Infof("[PLATFORM] Starting send via %s - Phone: %s, Device: %s, Has Image: %v", 
		platform, phone, deviceID, imageURL != "")
	logrus.Debugf("[PLATFORM] Message preview (first 100 chars): %s", truncateString(message, 100))
	
	// NO ANTI-SPAM HERE - Already handled by BroadcastWorker
	
	startTime := time.Now()
	var err error
	
	switch platform {
	case "Wablas":
		logrus.Infof("[WABLAS] Sending to %s with token: %s...", phone, truncateString(instance, 10))
		err = ps.sendViaWablas(instance, phone, message, imageURL)
	case "Whacenter":
		logrus.Infof("[WHACENTER] Sending to %s with device: %s", phone, instance)
		err = ps.sendViaWhacenter(instance, phone, message, imageURL)
	default:
		err = fmt.Errorf("unknown platform: %s", platform)
	}
	
	duration := time.Since(startTime)
	
	if err != nil {
		logrus.Errorf("[PLATFORM] ❌ FAILED sending via %s to %s - Error: %v (took %v)", 
			platform, phone, err, duration)
		return err
	}
	
	logrus.Infof("[PLATFORM] ✅ SUCCESS sending via %s to %s (took %v)", 
		platform, phone, duration)
	return nil
}

// sendViaWablas sends message via Wablas API
func (ps *PlatformSender) sendViaWablas(token, phone, message, imageURL string) error {
	if imageURL != "" {
		// Send image with caption
		return ps.sendWablasImage(token, phone, message, imageURL)
	}
	// Send text only
	return ps.sendWablasText(token, phone, message)
}

// sendWablasText sends text message via Wablas
func (ps *PlatformSender) sendWablasText(token, phone, message string) error {
	apiURL := "https://my.wablas.com/api/send-message"
	
	logrus.Infof("[WABLAS-TEXT] Preparing request to %s for phone: %s", apiURL, phone)
	
	// Prepare form data
	data := url.Values{}
	data.Set("phone", phone)
	data.Set("message", message)
	
	logrus.Debugf("[WABLAS-TEXT] Request data: phone=%s, message_length=%d", phone, len(message))
	
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		logrus.Errorf("[WABLAS-TEXT] Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	logrus.Debugf("[WABLAS-TEXT] Sending POST request with Authorization token")
	
	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		logrus.Errorf("[WABLAS-TEXT] HTTP request failed: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("[WABLAS-TEXT] Failed to read response body: %v", err)
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	logrus.Infof("[WABLAS-TEXT] Response Status: %d", resp.StatusCode)
	logrus.Infof("[WABLAS-TEXT] Response Body: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("[WABLAS-TEXT] ❌ API Error - Status: %d, Body: %s", resp.StatusCode, string(body))
		return fmt.Errorf("wablas API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	logrus.Infof("[WABLAS-TEXT] ✅ Message sent successfully to %s", phone)
	return nil
}

// sendWablasImage sends image message via Wablas
func (ps *PlatformSender) sendWablasImage(token, phone, caption, imageURL string) error {
	apiURL := "https://my.wablas.com/api/send-image"
	
	logrus.Infof("[WABLAS-IMAGE] Preparing request to %s for phone: %s", apiURL, phone)
	logrus.Debugf("[WABLAS-IMAGE] Image URL: %s", imageURL)
	
	// Prepare form data
	data := url.Values{}
	data.Set("phone", phone)
	data.Set("image", imageURL)
	data.Set("caption", caption)
	
	logrus.Debugf("[WABLAS-IMAGE] Request data: phone=%s, image=%s, caption_length=%d", 
		phone, truncateString(imageURL, 50), len(caption))
	
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		logrus.Errorf("[WABLAS-IMAGE] Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	logrus.Debugf("[WABLAS-IMAGE] Sending POST request with Authorization token")
	
	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		logrus.Errorf("[WABLAS-IMAGE] HTTP request failed: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("[WABLAS-IMAGE] Failed to read response body: %v", err)
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	logrus.Infof("[WABLAS-IMAGE] Response Status: %d", resp.StatusCode)
	logrus.Infof("[WABLAS-IMAGE] Response Body: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("[WABLAS-IMAGE] ❌ API Error - Status: %d, Body: %s", resp.StatusCode, string(body))
		return fmt.Errorf("wablas API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	logrus.Infof("[WABLAS-IMAGE] ✅ Image sent successfully to %s", phone)
	return nil
}

// sendViaWhacenter sends message via Whacenter API
func (ps *PlatformSender) sendViaWhacenter(deviceID, phone, message, imageURL string) error {
	apiURL := "https://api.whacenter.com/api/send"
	
	logrus.Infof("[WHACENTER] Preparing request to %s for phone: %s", apiURL, phone)
	
	// Prepare payload
	payload := map[string]interface{}{
		"device_id": deviceID,
		"number":    phone,
		"message":   message,
	}
	
	// Add image if provided
	if imageURL != "" {
		payload["file"] = imageURL
		logrus.Debugf("[WHACENTER] Including image: %s", truncateString(imageURL, 50))
	}
	
	logrus.Debugf("[WHACENTER] Request payload: device=%s, number=%s, message_length=%d, has_file=%v", 
		deviceID, phone, len(message), imageURL != "")
	
	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logrus.Errorf("[WHACENTER] Failed to marshal payload: %v", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	logrus.Debugf("[WHACENTER] JSON payload: %s", truncateString(string(jsonData), 200))
	
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.Errorf("[WHACENTER] Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
	logrus.Debugf("[WHACENTER] Sending POST request")
	
	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		logrus.Errorf("[WHACENTER] HTTP request failed: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("[WHACENTER] Failed to read response body: %v", err)
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	logrus.Infof("[WHACENTER] Response Status: %d", resp.StatusCode)
	logrus.Infof("[WHACENTER] Response Body: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("[WHACENTER] ❌ API Error - Status: %d, Body: %s", resp.StatusCode, string(body))
		return fmt.Errorf("whacenter API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	logrus.Infof("[WHACENTER] ✅ Message sent successfully to %s", phone)
	return nil
}

// truncateString truncates a string to specified length for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
