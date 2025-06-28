package broadcast

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	domainBroadcast "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/broadcast"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// DeviceWorker handles broadcasting for a single device
type DeviceWorker struct {
	deviceID        string
	client          *whatsmeow.Client
	minDelay        int
	maxDelay        int
	messageQueue    chan domainBroadcast.BroadcastMessage
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	status          string
	processedCount  int
	failedCount     int
	lastActivity    time.Time
	broadcastRepo   *repository.BroadcastRepository
}

// NewDeviceWorker creates a new device worker
func NewDeviceWorker(deviceID string, client *whatsmeow.Client, minDelay, maxDelay int) *DeviceWorker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DeviceWorker{
		deviceID:      deviceID,
		client:        client,
		minDelay:      minDelay,
		maxDelay:      maxDelay,
		messageQueue:  make(chan domainBroadcast.BroadcastMessage, 1000),
		ctx:           ctx,
		cancel:        cancel,
		status:        "idle",
		lastActivity:  time.Now(),
		broadcastRepo: repository.GetBroadcastRepository(),
	}
}

// Start starts the worker
func (dw *DeviceWorker) Start() {
	go dw.processMessages()
	go dw.healthCheck()
}

// Stop stops the worker
func (dw *DeviceWorker) Stop() {
	dw.mu.Lock()
	defer dw.mu.Unlock()
	
	if dw.cancel != nil {
		dw.cancel()
	}
	close(dw.messageQueue)
	dw.status = "stopped"
}

// QueueMessage adds a message to the worker's queue
func (dw *DeviceWorker) QueueMessage(msg domainBroadcast.BroadcastMessage) error {
	select {
	case dw.messageQueue <- msg:
		return nil
	case <-time.After(time.Second * 5):
		return fmt.Errorf("queue full for device %s", dw.deviceID)
	}
}

// GetStatus returns worker status
func (dw *DeviceWorker) GetStatus() domainBroadcast.WorkerStatus {
	dw.mu.RLock()
	defer dw.mu.RUnlock()
	
	return domainBroadcast.WorkerStatus{
		DeviceID:       dw.deviceID,
		Status:         dw.status,
		QueueSize:      len(dw.messageQueue),
		ProcessedCount: dw.processedCount,
		FailedCount:    dw.failedCount,
		LastActivity:   dw.lastActivity,
	}
}

// processMessages processes messages from the queue
func (dw *DeviceWorker) processMessages() {
	for {
		select {
		case <-dw.ctx.Done():
			return
		case msg, ok := <-dw.messageQueue:
			if !ok {
				return
			}
			
			dw.mu.Lock()
			dw.status = "processing"
			dw.lastActivity = time.Now()
			dw.mu.Unlock()
			
			// Process the message
			err := dw.sendMessage(msg)
			
			dw.mu.Lock()
			if err != nil {
				dw.failedCount++
				logrus.Errorf("Failed to send message: %v", err)
				// Update broadcast status to failed
				if msg.ID != "" {
					dw.broadcastRepo.UpdateMessageStatus(msg.ID, "failed", err.Error())
				}
			} else {
				dw.processedCount++
				// Update broadcast status to sent
				if msg.ID != "" {
					dw.broadcastRepo.UpdateMessageStatus(msg.ID, "sent", "")
					logrus.Infof("Message %s sent successfully to %s", msg.ID, msg.RecipientPhone)
				}
			}
			dw.status = "idle"
			dw.mu.Unlock()
			
			// Determine delay based on message's campaign/sequence settings
			var delay time.Duration
			
			// Use delays from the message (set by campaign/sequence)
			if msg.MinDelay > 0 || msg.MaxDelay > 0 {
				delay = getRandomDelayBetween(msg.MinDelay, msg.MaxDelay)
				logrus.Debugf("Using campaign/sequence delays: %d-%d seconds", msg.MinDelay, msg.MaxDelay)
			} else {
				// Fallback to device defaults if message doesn't have delays
				delay = dw.getRandomDelay()
				logrus.Debugf("Using device default delays: %d-%d seconds", dw.minDelay, dw.maxDelay)
			}
			
			// Log the message processing
			logrus.Infof("Processing message for %s - Type: %s, Has image: %v, Has text: %v, Delay: %v", 
				msg.RecipientPhone, msg.Type, msg.MediaURL != "", msg.Content != "", delay)
			
			time.Sleep(delay)
		}
	}
}

// sendMessage sends a message based on type
func (dw *DeviceWorker) sendMessage(msg domainBroadcast.BroadcastMessage) error {
	// Parse recipient JID
	recipient, err := whatsapp.ParseJID(msg.RecipientPhone)
	if err != nil {
		return fmt.Errorf("invalid recipient: %v", err)
	}
	
	switch msg.Type {
	case "text":
		return dw.sendTextMessage(recipient, msg)
	case "image":
		return dw.sendImageMessage(recipient, msg)
	case "video":
		return dw.sendVideoMessage(recipient, msg)
	case "document":
		return dw.sendDocumentMessage(recipient, msg)
	default:
		return fmt.Errorf("unsupported message type: %s", msg.Type)
	}
}

// sendTextMessage sends text message
func (dw *DeviceWorker) sendTextMessage(recipient types.JID, msg domainBroadcast.BroadcastMessage) error {
	message := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: &msg.Content,
		},
	}
	
	_, err := dw.client.SendMessage(context.Background(), recipient, message)
	return err
}

// sendImageMessage sends image message with caption
func (dw *DeviceWorker) sendImageMessage(recipient types.JID, msg domainBroadcast.BroadcastMessage) error {
	// Download image from URL
	imageData, err := downloadMedia(msg.MediaURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %v", err)
	}
	
	// Upload to WhatsApp servers
	uploaded, err := dw.client.Upload(context.Background(), imageData, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image: %v", err)
	}
	
	// Create image message
	message := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       &msg.Content,
			URL:           &uploaded.URL,
			DirectPath:    &uploaded.DirectPath,
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    &uploaded.FileLength,
			Mimetype:      proto.String("image/jpeg"),
		},
	}
	
	_, err = dw.client.SendMessage(context.Background(), recipient, message)
	return err
}

// sendVideoMessage sends video message
func (dw *DeviceWorker) sendVideoMessage(recipient types.JID, msg domainBroadcast.BroadcastMessage) error {
	// TODO: Implement video sending
	return fmt.Errorf("video messages not yet implemented")
}

// sendDocumentMessage sends document message
func (dw *DeviceWorker) sendDocumentMessage(recipient types.JID, msg domainBroadcast.BroadcastMessage) error {
	// TODO: Implement document sending
	return fmt.Errorf("document messages not yet implemented")
}

// getRandomDelay returns a random delay between min and max
func (dw *DeviceWorker) getRandomDelay() time.Duration {
	if dw.minDelay == dw.maxDelay {
		return time.Duration(dw.minDelay) * time.Second
	}
	
	delay := rand.Intn(dw.maxDelay-dw.minDelay) + dw.minDelay
	return time.Duration(delay) * time.Second
}

// healthCheck monitors worker health
func (dw *DeviceWorker) healthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-dw.ctx.Done():
			return
		case <-ticker.C:
			dw.mu.RLock()
			lastActivity := dw.lastActivity
			status := dw.status
			dw.mu.RUnlock()
			
			// Check if worker is stuck
			if status == "processing" && time.Since(lastActivity) > 10*time.Minute {
				logrus.Warnf("Worker %s appears stuck, restarting...", dw.deviceID)
				// TODO: Implement restart logic
			}
		}
	}
}

// SendMessage sends a message (implements direct sending without queue)
func (dw *DeviceWorker) SendMessage(msg domainBroadcast.BroadcastMessage) error {
	return dw.sendMessage(msg)
}

// Run starts the worker processes
func (dw *DeviceWorker) Run() {
	dw.Start()
}

// IsHealthy checks if the worker is healthy
func (dw *DeviceWorker) IsHealthy() bool {
	dw.mu.RLock()
	defer dw.mu.RUnlock()
	
	// Check if stuck processing
	if dw.status == "processing" && time.Since(dw.lastActivity) > 10*time.Minute {
		return false
	}
	
	// Check if stopped
	if dw.status == "stopped" {
		return false
	}
	
	// Check if client is still connected
	if dw.client == nil || !dw.client.IsConnected() {
		return false
	}
	
	return true
}

// downloadMedia downloads media from URL
func downloadMedia(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}
	
	return io.ReadAll(resp.Body)
}
// getRandomDelayBetween returns a random delay between min and max seconds
func getRandomDelayBetween(minDelay, maxDelay int) time.Duration {
	if minDelay <= 0 && maxDelay <= 0 {
		// Default delays if not set
		minDelay = 10
		maxDelay = 30
	}
	
	if minDelay == maxDelay || maxDelay <= minDelay {
		return time.Duration(minDelay) * time.Second
	}
	
	delay := rand.Intn(maxDelay-minDelay) + minDelay
	return time.Duration(delay) * time.Second
}
