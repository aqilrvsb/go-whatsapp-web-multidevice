package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
			Timeout: 120 * time.Second, // Increased from 30s to 120s for slow APIs
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 30 * time.Second, // Increased TLS handshake timeout
			},
		},
		messageRandomizer: antipattern.NewMessageRandomizer(),
		greetingProcessor: antipattern.NewGreetingProcessor(),
	}
}

// retryWithBackoff retries a function with exponential backoff
func (ps *PlatformSender) retryWithBackoff(fn func() error, maxRetries int, platform string) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Try the function
		err := fn()
		if err == nil {
			// Success!
			if attempt > 0 {
				logrus.Infof("[%s] Request succeeded after %d retries", platform, attempt)
			}
			return nil
		}

		lastErr = err

		// If this was the last attempt, return the error
		if attempt == maxRetries {
			logrus.Errorf("[%s] Failed after %d retries: %v", platform, maxRetries, err)
			return lastErr
		}

		// Check if error is retryable
		errStr := err.Error()
		isRetryable := strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "TLS handshake") ||
			strings.Contains(errStr, "EOF") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "broken pipe")

		if !isRetryable {
			// Don't retry for non-network errors
			logrus.Warnf("[%s] Non-retryable error: %v", platform, err)
			return err
		}

		// Calculate backoff delay: 2^attempt seconds (2s, 4s, 8s)
		backoffDelay := time.Duration(1<<uint(attempt)) * 2 * time.Second
		logrus.Warnf("[%s] Attempt %d failed: %v. Retrying in %v...", platform, attempt+1, err, backoffDelay)
		time.Sleep(backoffDelay)
	}

	return lastErr
}

// SendMessage sends a message via external platform
// NOTE: Anti-spam is already applied by BroadcastWorker - we just send raw content
func (ps *PlatformSender) SendMessage(platform, instance, phone, recipientName, message, imageURL, deviceID string) error {
	// NO ANTI-SPAM HERE - Already handled by BroadcastWorker

	_ = time.Now() // startTime - for future use
	var err error

	switch platform {
	case "Wablas":
		err = ps.sendViaWablas(instance, phone, message, imageURL)
	case "Whacenter":
		// Retry Whacenter with backoff (3 retries max)
		err = ps.retryWithBackoff(func() error {
			return ps.sendViaWhacenter(instance, phone, message, imageURL)
		}, 3, "Whacenter")
	default:
		err = fmt.Errorf("unknown platform: %s", platform)
	}

	if err != nil {
		return err
	}

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

	// Format message for clean WhatsApp display
	message = formatWhatsAppMessage(message)

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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Check status
	if status, ok := result["status"].(bool); ok && !status {
		if msg, ok := result["message"].(string); ok {
			return fmt.Errorf("wablas error: %s", msg)
		}
		return fmt.Errorf("wablas returned false status: %s", string(body))
	}
	
	return nil
}

// sendWablasImage sends image with caption via Wablas
func (ps *PlatformSender) sendWablasImage(token, phone, caption, imageURL string) error {
	apiURL := "https://my.wablas.com/api/send-image"

	// Format caption for clean WhatsApp display
	caption = formatWhatsAppMessage(caption)

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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Check status
	if status, ok := result["status"].(bool); ok && !status {
		if msg, ok := result["message"].(string); ok {
			return fmt.Errorf("wablas error: %s", msg)
		}
		return fmt.Errorf("wablas returned false status")
	}
	
	return nil
}

// sendViaWhacenter sends message via Whacenter API
func (ps *PlatformSender) sendViaWhacenter(deviceID, phone, message, imageURL string) error {
	apiURL := "https://api.whacenter.com/api/send"

	// Format phone number - ALWAYS use Malaysia country code (60)
	// Remove any existing country code prefix and force 60
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.TrimPrefix(phone, "60")  // Remove 60 if exists
	phone = strings.TrimPrefix(phone, "62")  // Remove 62 if exists (Indonesia)
	phone = strings.TrimPrefix(phone, "0")   // Remove leading 0 if exists
	phone = "60" + phone  // Always add Malaysian country code 60

	// Format message for clean WhatsApp display
	message = formatWhatsAppMessage(message)
	logrus.Debugf("WhatsCenter formatted message preview: %s", truncateString(message, 100))

	// Create form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	// Add fields
	writer.WriteField("device_id", deviceID)
	writer.WriteField("number", phone)
	writer.WriteField("message", message)
	
	// Add image if provided
	if imageURL != "" {
		writer.WriteField("file", imageURL)
	}
	
	writer.Close()
	
	// Create request
	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// Send request
	resp, err := ps.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Log the raw response for debugging
	logrus.Debugf("WhatsCenter raw response: %s", string(respBody))
	
	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w, body: %s", err, string(respBody))
	}
	
	// Log parsed response
	logrus.Debugf("WhatsCenter parsed response: %+v", result)
	
	// Check status
	if status, ok := result["status"].(bool); ok && !status {
		if msg, ok := result["msg"].(string); ok {
			logrus.Errorf("WhatsCenter error - status: false, msg: %s, full response: %+v", msg, result)
			return fmt.Errorf("whacenter error: %s", msg)
		}
		logrus.Errorf("WhatsCenter error - status: false, no msg field, full response: %+v", result)
		return fmt.Errorf("whacenter returned false status, response: %+v", result)
	}
	
	return nil
}

// formatWhatsAppMessage formats message for clean WhatsApp display
// Adds proper line breaks, spacing, and makes it easy to read
func formatWhatsAppMessage(message string) string {
	// Step 1: Normalize line breaks (handle \r\n, \n, etc.)
	message = strings.ReplaceAll(message, "\r\n", "\n")
	message = strings.ReplaceAll(message, "\r", "\n")

	// Step 2: Fix spacing around emojis
	// Add space after emoji if missing (e.g., "Kak😊, Saya" -> "Kak 😊, Saya")
	message = strings.ReplaceAll(message, "😊,", "😊\n\n")
	message = strings.ReplaceAll(message, "😊.", "😊\n\n")
	message = strings.ReplaceAll(message, "🙂,", "🙂\n\n")
	message = strings.ReplaceAll(message, "🙂.", "🙂\n\n")

	// Step 3: Add line breaks after sentences ending with . ? !
	// But only if not already followed by line break
	message = strings.ReplaceAll(message, ". ", ".\n\n")
	message = strings.ReplaceAll(message, "? ", "?\n\n")
	message = strings.ReplaceAll(message, "! ", "!\n\n")

	// Step 4: Remove excessive line breaks (more than 2 consecutive)
	for strings.Contains(message, "\n\n\n") {
		message = strings.ReplaceAll(message, "\n\n\n", "\n\n")
	}

	// Step 5: Remove leading/trailing whitespace
	message = strings.TrimSpace(message)

	// Step 6: Remove multiple spaces
	for strings.Contains(message, "  ") {
		message = strings.ReplaceAll(message, "  ", " ")
	}

	return message
}

// truncateString truncates a string to specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}