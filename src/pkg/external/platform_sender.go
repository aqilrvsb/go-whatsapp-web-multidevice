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
	
	"github.com/sirupsen/logrus"
)

// PlatformSender handles sending messages via external platforms
type PlatformSender struct {
	client *http.Client
}

// NewPlatformSender creates a new platform sender
func NewPlatformSender() *PlatformSender {
	return &PlatformSender{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendMessage sends a message via external platform
func (ps *PlatformSender) SendMessage(platform, instance, phone, message, imageURL string) error {
	switch platform {
	case "Wablas":
		return ps.sendViaWablas(instance, phone, message, imageURL)
	case "Whacenter":
		return ps.sendViaWhacenter(instance, phone, message, imageURL)
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}
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
	
	// Prepare form data
	data := url.Values{}
	data.Set("phone", phone)
	data.Set("message", message)
	
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	logrus.Infof("Wablas text response: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wablas API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// sendWablasImage sends image message via Wablas
func (ps *PlatformSender) sendWablasImage(token, phone, caption, imageURL string) error {
	apiURL := "https://my.wablas.com/api/send-image"
	
	// Prepare form data
	data := url.Values{}
	data.Set("phone", phone)
	data.Set("image", imageURL)
	data.Set("caption", caption)
	
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	logrus.Infof("Wablas image response: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wablas API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// sendViaWhacenter sends message via Whacenter API
func (ps *PlatformSender) sendViaWhacenter(deviceID, phone, message, imageURL string) error {
	apiURL := "https://api.whacenter.com/api/send"
	
	// Prepare payload
	payload := map[string]interface{}{
		"device_id": deviceID,
		"number":    phone,
		"message":   message,
	}
	
	// Add image if provided
	if imageURL != "" {
		payload["file"] = imageURL
	}
	
	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	logrus.Infof("Whacenter response: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("whacenter API error: status %d, body: %s", resp.StatusCode, string(body))
	}
	
	return nil
}
