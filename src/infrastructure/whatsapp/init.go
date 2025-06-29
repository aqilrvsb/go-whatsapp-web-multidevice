package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Type definitions
type ExtractedMedia struct {
	MediaPath string `json:"media_path"`
	MimeType  string `json:"mime_type"`
	Caption   string `json:"caption"`
}

type evtReaction struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

type evtMessage struct {
	ID            string `json:"id,omitempty"`
	Text          string `json:"text,omitempty"`
	RepliedId     string `json:"replied_id,omitempty"`
	QuotedMessage string `json:"quoted_message,omitempty"`
}

// Global variables
var (
	cli           *whatsmeow.Client
	log           waLog.Logger
	historySyncID int32
	startupTime   = time.Now().Unix()
)

// InitWaDB initializes the WhatsApp database connection
func InitWaDB(ctx context.Context) *sqlstore.Container {
	log = waLog.Stdout("Main", config.WhatsappLogLevel, true)
	dbLog := waLog.Stdout("Database", config.WhatsappLogLevel, true)

	storeContainer, err := initDatabase(ctx, dbLog)
	if err != nil {
		log.Errorf("Database initialization error: %v", err)
		panic(pkgError.InternalServerError(fmt.Sprintf("Database initialization error: %v", err)))
	}

	return storeContainer
}

// initDatabase creates and returns a database store container based on the configured URI
func initDatabase(ctx context.Context, dbLog waLog.Logger) (*sqlstore.Container, error) {
	if strings.HasPrefix(config.DBURI, "file:") {
		return sqlstore.New(ctx, "sqlite3", config.DBURI, dbLog)
	} else if strings.HasPrefix(config.DBURI, "postgres:") || strings.HasPrefix(config.DBURI, "postgresql:") {
		// Convert postgresql:// to postgres:// for the driver
		dbUri := config.DBURI
		if strings.HasPrefix(dbUri, "postgresql://") {
			dbUri = strings.Replace(dbUri, "postgresql://", "postgres://", 1)
		}
		return sqlstore.New(ctx, "postgres", dbUri, dbLog)
	}

	return nil, fmt.Errorf("unknown database type: %s. Currently only sqlite3(file:) and postgres/postgresql are supported", config.DBURI)
}

// InitWaCLI initializes the WhatsApp client
func InitWaCLI(ctx context.Context, storeContainer *sqlstore.Container) *whatsmeow.Client {
	device, err := storeContainer.GetFirstDevice(ctx)
	if err != nil {
		log.Errorf("Failed to get device: %v", err)
		panic(err)
	}

	if device == nil {
		log.Infof("No device found - devices will be created when users add them")
		// For multi-device support, return nil instead of panicking
		// Devices will be created dynamically when users add them
		return nil
	}

	// Configure device properties
	osName := fmt.Sprintf("%s %s", config.AppOs, config.AppVersion)
	store.DeviceProps.PlatformType = &config.AppPlatform
	store.DeviceProps.Os = &osName

	// Create and configure the client
	cli = whatsmeow.NewClient(device, waLog.Stdout("Client", config.WhatsappLogLevel, true))
	cli.EnableAutoReconnect = true
	cli.AutoTrustIdentity = true
	cli.AddEventHandler(func(rawEvt interface{}) {
		handler(ctx, rawEvt)
	})
	
	// Set as global client for debugging
	SetGlobalClient(cli)

	return cli
}

// handler is the main event handler for WhatsApp events
func handler(ctx context.Context, rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.DeleteForMe:
		handleDeleteForMe(ctx, evt)
	case *events.AppStateSyncComplete:
		handleAppStateSyncComplete(ctx, evt)
	case *events.PairSuccess:
		handlePairSuccess(ctx, evt)
	case *events.LoggedOut:
		handleLoggedOut(ctx)
	case *events.Connected, *events.PushNameSetting:
		handleConnectionEvents(ctx)
	case *events.StreamReplaced:
		handleStreamReplaced(ctx)
	case *events.Message:
		handleMessage(ctx, evt)
	case *events.Receipt:
		handleReceipt(ctx, evt)
	case *events.Presence:
		handlePresence(ctx, evt)
	case *events.HistorySync:
		handleHistorySync(ctx, evt)
	case *events.AppState:
		handleAppState(ctx, evt)
	}
}

// Event handler functions

func handleDeleteForMe(_ context.Context, evt *events.DeleteForMe) {
	log.Infof("Deleted message %s for %s", evt.MessageID, evt.SenderJID.String())
}

func handleAppStateSyncComplete(_ context.Context, evt *events.AppStateSyncComplete) {
	if len(cli.Store.PushName) > 0 && evt.Name == appstate.WAPatchCriticalBlock {
		if err := cli.SendPresence(types.PresenceAvailable); err != nil {
			log.Warnf("Failed to send available presence: %v", err)
		} else {
			log.Infof("Marked self as available")
		}
	}
}

func handlePairSuccess(_ context.Context, evt *events.PairSuccess) {
	log.Infof("Pair success! ID: %s, BusinessName: %s, Platform: %s", 
		evt.ID.String(), evt.BusinessName, evt.Platform)
	
	// Log the phone number for debugging
	if evt.ID.User != "" {
		log.Infof("Phone number from pair success: %s", evt.ID.User)
		
		// Try to find and update the device immediately
		userRepo := repository.GetUserRepository()
		for userID, session := range connectionSessions {
			if session != nil && session.DeviceID != "" {
				log.Infof("Found session for user %s, updating device %s", userID, session.DeviceID)
				// Update with phone number from pair success
				err := userRepo.UpdateDeviceStatus(session.DeviceID, "connecting", evt.ID.User, evt.ID.String())
				if err != nil {
					log.Errorf("Failed to update device on pair success: %v", err)
				}
				break
			}
		}
	}
	
	// Update device status in database
	if userRepo := repository.GetUserRepository(); userRepo != nil {
		// TODO: Get current user and device ID from context
		// For now, just broadcast the success
		log.Infof("Device paired successfully, waiting for full connection...")
	}
	
	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:    "LOGIN_SUCCESS",
		Message: fmt.Sprintf("Successfully paired with %s", evt.ID.String()),
	}
	
	// The device is paired but not fully connected yet
	// Wait for Connected event to update status to "online"
}

func handleLoggedOut(_ context.Context) {
	websocket.Broadcast <- websocket.BroadcastMessage{
		Code:   "LIST_DEVICES",
		Result: nil,
	}
}

func handleConnectionEvents(_ context.Context) {
	log.Infof("WhatsApp connection event received")
	
	if !cli.IsConnected() {
		log.Warnf("Connection event received but client not connected")
		return
	}
	
	if cli.IsLoggedIn() {
		log.Infof("WhatsApp client is logged in and connected!")
		
		// Get device info for database update
		var phoneNumber, jid string
		var connectedDeviceID string // Declare here for broader scope
		
		if cli.Store.ID != nil {
			jid = cli.Store.ID.String()
			phoneNumber = cli.Store.ID.User
			log.Infof("Connected as: %s (Phone: %s, Name: %s)", jid, phoneNumber, cli.Store.PushName)
			
			// Update device status in database
			userRepo := repository.GetUserRepository()
			
			// Look for any active connection session
			allSessions := GetAllConnectionSessions()
			log.Infof("Found %d active connection sessions", len(allSessions))
			for userID, session := range allSessions {
				if session != nil && session.DeviceID != "" {
					log.Infof("Updating device status for user %s, device %s", userID, session.DeviceID)
					connectedDeviceID = session.DeviceID // Store device ID for the message
					
					// Update device status to online
					err := userRepo.UpdateDeviceStatus(session.DeviceID, "online", phoneNumber, jid)
					if err != nil {
						log.Errorf("Failed to update device status: %v", err)
					} else {
						log.Infof("Successfully updated device %s to online status", session.DeviceID)
						log.Infof("Also updating with phone: %s", phoneNumber)
						
						// Register device with client manager using the device ID from database
						cm := GetClientManager()
						cm.AddClient(session.DeviceID, cli)
						log.Infof("Registered device %s with client manager for broadcast system", session.DeviceID)
						
						// Trigger initial chat sync
						go func() {
							time.Sleep(3 * time.Second) // Wait for connection to stabilize
							chats, err := GetChatsForDevice(session.DeviceID)
							if err != nil {
								log.Errorf("Failed to sync chats for device %s: %v", session.DeviceID, err)
							} else {
								log.Infof("Successfully synced %d chats for device %s", len(chats), session.DeviceID)
							}
						}()
					}
					
					// Clear the session after successful update
					ClearConnectionSession(userID)
					break
				}
			}
			
			// If we didn't find device ID from session, try to find it by phone/JID
			if connectedDeviceID == "" {
				log.Infof("No device ID found in session, attempting to find device by phone: %s", phoneNumber)
				
				// Log all devices for debugging
				var debugQuery = `SELECT id, COALESCE(phone, ''), device_name FROM user_devices WHERE status != 'deleted'`
				rows, _ := userRepo.DB().Query(debugQuery)
				if rows != nil {
					defer rows.Close()
					log.Infof("=== All devices in database ===")
					for rows.Next() {
						var id, phone, name string
						if err := rows.Scan(&id, &phone, &name); err != nil {
							log.Errorf("Error scanning row: %v", err)
							continue
						}
						log.Infof("Device: %s, Phone: '%s', Name: %s", id, phone, name)
					}
					log.Infof("=== End of devices ===")
					log.Infof("Looking for phone: '%s'", phoneNumber)
				}
				
				// Try to find the device by phone number
				query := `SELECT id FROM user_devices WHERE phone = $1 AND status != 'deleted' LIMIT 1`
				err := userRepo.DB().QueryRow(query, phoneNumber).Scan(&connectedDeviceID)
				
				if err == nil && connectedDeviceID != "" {
					log.Infof("Found device ID from database by phone: %s", connectedDeviceID)
					
					// Update device status to online and update JID
					err = userRepo.UpdateDeviceStatus(connectedDeviceID, "online", phoneNumber, jid)
					if err != nil {
						log.Errorf("Failed to update device status: %v", err)
					} else {
						log.Infof("Successfully updated device %s to online status (found by phone)", connectedDeviceID)
						
						// Register device with client manager
						cm := GetClientManager()
						cm.AddClient(connectedDeviceID, cli)
						log.Infof("Registered device %s with client manager for broadcast system", connectedDeviceID)
					}
				} else {
					log.Warnf("Could not find device by phone %s: %v", phoneNumber, err)
					log.Infof("Devices should be pre-registered with the exact phone number that WhatsApp returns")
					log.Infof("WhatsApp returned phone: %s, but no matching device found", phoneNumber)
				}
			}
		}
		
		// Send connection success message with device ID
		websocket.Broadcast <- websocket.BroadcastMessage{
			Code:    "DEVICE_CONNECTED",
			Message: "WhatsApp fully connected and logged in",
			Result: map[string]interface{}{
				"phone":    phoneNumber,
				"jid":      jid,
				"deviceId": connectedDeviceID,
			},
		}
	}
	
	if len(cli.Store.PushName) == 0 {
		return
	}

	// Send presence available when connecting and when the pushname is changed.
	// This makes sure that outgoing messages always have the right pushname.
	if err := cli.SendPresence(types.PresenceAvailable); err != nil {
		log.Warnf("Failed to send available presence: %v", err)
	} else {
		log.Infof("Marked self as available")
	}
}

func handleStreamReplaced(_ context.Context) {
	os.Exit(0)
}

func handleMessage(ctx context.Context, evt *events.Message) {
	// Log message metadata
	metaParts := buildMessageMetaParts(evt)
	log.Infof("Received message %s from %s (%s): %+v",
		evt.Info.ID,
		evt.Info.SourceString(),
		strings.Join(metaParts, ", "),
		evt.Message,
	)

	// Record the message with analytics
	message := ExtractMessageText(evt)
	utils.RecordMessage(evt.Info.ID, evt.Info.Sender.String(), message)
	
	log.Infof("WhatsappChatStorage enabled: %v", config.WhatsappChatStorage)
	
	// Save message to WhatsApp storage for all connected devices
	if config.WhatsappChatStorage {
		cm := GetClientManager()
		allClients := cm.GetAllClients()
		for deviceID, client := range allClients {
		// Skip group messages - we only want personal chats
		if evt.Info.IsGroup || evt.Info.Chat.Server == types.GroupServer {
			continue
		}
		
		// Skip broadcast and status messages
		if evt.Info.Chat.Server == types.BroadcastServer || evt.Info.Chat.User == "status" {
			continue
		}
		
		// Only process personal chats
		if evt.Info.Chat.Server != types.DefaultUserServer {
			continue
		}
		// Check if this message belongs to this client's conversation
		// Either sent by this client OR sent to this client
		if client.Store.ID != nil {
			// For personal chats, we save all messages
			// In personal chats, if I didn't send it, then it was sent to me
			isPersonalChat := evt.Info.Chat.Server == types.DefaultUserServer && !evt.Info.IsGroup
			
			// Save all messages in personal chats
			if isPersonalChat {
				log.Infof("Saving message for device %s: sender=%s, chat=%s, isFromMe=%v",
					deviceID, evt.Info.Sender.String(), evt.Info.Chat.String(), evt.Info.IsFromMe)
			// Get sender name
			senderName := ""
			if evt.Info.IsFromMe {
				senderName = "You"
			} else {
				contact, _ := client.Store.Contacts.GetContact(context.Background(), evt.Info.Sender)
				if contact.Found && contact.PushName != "" {
					senderName = contact.PushName
				} else {
					senderName = evt.Info.Sender.User
				}
			}
			
			// Determine message type
			messageType := "text"
			mediaURL := ""
			if evt.Message.ImageMessage != nil {
				messageType = "image"
			} else if evt.Message.VideoMessage != nil {
				messageType = "video"
			} else if evt.Message.AudioMessage != nil {
				messageType = "audio"
			} else if evt.Message.DocumentMessage != nil {
				messageType = "document"
			}
			
			// Save message to database
			whatsappRepo := repository.GetWhatsAppRepository()
			whatsappMsg := repository.WhatsAppMessage{
				DeviceID:    deviceID,
				ChatJID:     evt.Info.Chat.String(),
				MessageID:   evt.Info.ID,
				SenderJID:   evt.Info.Sender.String(),
				SenderName:  senderName,
				MessageText: message,
				MessageType: messageType,
				MediaURL:    mediaURL,
				IsSent:      evt.Info.IsFromMe,
				IsRead:      false,
				Timestamp:   evt.Info.Timestamp,
			}
			
			if err := whatsappRepo.SaveMessage(&whatsappMsg); err != nil {
				log.Errorf("Failed to save message: %v", err)
			}
			
			// Update chat's last message
			chat, err := whatsappRepo.GetChatByJID(deviceID, evt.Info.Chat.String())
			if err != nil {
				// Create new chat entry
				chatName := ""
				if evt.Info.IsGroup {
					// Get group info
					groupInfo, _ := client.GetGroupInfo(evt.Info.Chat)
					if groupInfo != nil {
						chatName = groupInfo.Name
					}
				} else {
					// Get contact name
					contact, _ := client.Store.Contacts.GetContact(context.Background(), evt.Info.Chat)
					if contact.Found && contact.PushName != "" {
						chatName = contact.PushName
					} else {
						chatName = evt.Info.Chat.User
					}
				}
				
				chat = &repository.WhatsAppChat{
					DeviceID:        deviceID,
					ChatJID:         evt.Info.Chat.String(),
					ChatName:        chatName,
					IsGroup:         evt.Info.IsGroup,
					IsMuted:         false,
					LastMessageText: message,
					LastMessageTime: evt.Info.Timestamp,
					UnreadCount:     0,
				}
			} else {
				// Update existing chat
				chat.LastMessageText = message
				chat.LastMessageTime = evt.Info.Timestamp
				if !evt.Info.IsFromMe {
					chat.UnreadCount++
				}
			}
			
			if err := whatsappRepo.SaveOrUpdateChat(chat); err != nil {
				log.Errorf("Failed to update chat: %v", err)
			}
			
			break
			}
		}
	}
	}
	
	// Record to database for analytics
	// TODO: Get actual user and device from session context
	// For now, we'll need to implement a device mapping system
	
	// Determine message status
	status := "sent"
	if !evt.Info.IsFromMe {
		status = "received"
	}
	
	// Try to record in database (if we have user context)
	if analyticsRepo := ctx.Value("analyticsRepo"); analyticsRepo != nil {
		if repo, ok := analyticsRepo.(*repository.MessageAnalyticsRepository); ok {
			// Get user and device info from context
			userID := ""
			deviceID := ""
			
			if userCtx := ctx.Value("userID"); userCtx != nil {
				userID = userCtx.(string)
			}
			if deviceCtx := ctx.Value("deviceID"); deviceCtx != nil {
				deviceID = deviceCtx.(string)
			}
			
			if userID != "" && deviceID != "" {
				repo.RecordMessage(
					userID,
					deviceID,
					evt.Info.ID,
					evt.Info.Chat.String(),
					message,
					evt.Info.IsFromMe,
					status,
				)
			}
		}
	}

	// Handle image message if present
	handleImageMessage(ctx, evt)

	// Handle auto-reply if configured
	handleAutoReply(evt)

	// Forward to webhook if configured
	handleWebhookForward(ctx, evt)
}

func buildMessageMetaParts(evt *events.Message) []string {
	metaParts := []string{
		fmt.Sprintf("pushname: %s", evt.Info.PushName),
		fmt.Sprintf("timestamp: %s", evt.Info.Timestamp),
	}
	if evt.Info.Type != "" {
		metaParts = append(metaParts, fmt.Sprintf("type: %s", evt.Info.Type))
	}
	if evt.Info.Category != "" {
		metaParts = append(metaParts, fmt.Sprintf("category: %s", evt.Info.Category))
	}
	if evt.IsViewOnce {
		metaParts = append(metaParts, "view once")
	}
	return metaParts
}

func handleImageMessage(ctx context.Context, evt *events.Message) {
	if img := evt.Message.GetImageMessage(); img != nil {
		if path, err := ExtractMedia(ctx, config.PathStorages, img); err != nil {
			log.Errorf("Failed to download image: %v", err)
		} else {
			log.Infof("Image downloaded to %s", path)
		}
	}
}

func handleAutoReply(evt *events.Message) {
	if config.WhatsappAutoReplyMessage != "" &&
		!isGroupJid(evt.Info.Chat.String()) &&
		!evt.Info.IsIncomingBroadcast() &&
		evt.Message.GetExtendedTextMessage().GetText() != "" {
		_, _ = cli.SendMessage(
			context.Background(),
			FormatJID(evt.Info.Sender.String()),
			&waE2E.Message{Conversation: proto.String(config.WhatsappAutoReplyMessage)},
		)
	}
}

func handleWebhookForward(ctx context.Context, evt *events.Message) {
	if len(config.WhatsappWebhook) > 0 &&
		!strings.Contains(evt.Info.SourceString(), "broadcast") &&
		!isFromMySelf(evt.Info.SourceString()) {
		go func(evt *events.Message) {
			if err := forwardToWebhook(ctx, evt); err != nil {
				logrus.Error("Failed forward to webhook: ", err)
			}
		}(evt)
	}
}

func handleReceipt(ctx context.Context, evt *events.Receipt) {
	if evt.Type == types.ReceiptTypeRead || evt.Type == types.ReceiptTypeReadSelf {
		log.Infof("%v was read by %s at %s", evt.MessageIDs, evt.SourceString(), evt.Timestamp)
		// Update message status to "read" in database
		if analyticsRepo := ctx.Value("analyticsRepo"); analyticsRepo != nil {
			if repo, ok := analyticsRepo.(*repository.MessageAnalyticsRepository); ok {
				for _, msgID := range evt.MessageIDs {
					repo.UpdateMessageStatus(msgID, "read")
				}
			}
		}
	} else if evt.Type == types.ReceiptTypeDelivered {
		log.Infof("%s was delivered to %s at %s", evt.MessageIDs[0], evt.SourceString(), evt.Timestamp)
		// Update message status to "delivered" in database
		if analyticsRepo := ctx.Value("analyticsRepo"); analyticsRepo != nil {
			if repo, ok := analyticsRepo.(*repository.MessageAnalyticsRepository); ok {
				for _, msgID := range evt.MessageIDs {
					repo.UpdateMessageStatus(msgID, "delivered")
				}
			}
		}
	}
}

func handlePresence(_ context.Context, evt *events.Presence) {
	if evt.Unavailable {
		if evt.LastSeen.IsZero() {
			log.Infof("%s is now offline", evt.From)
		} else {
			log.Infof("%s is now offline (last seen: %s)", evt.From, evt.LastSeen)
		}
	} else {
		log.Infof("%s is now online", evt.From)
	}
}

func handleHistorySync(_ context.Context, evt *events.HistorySync) {
	id := atomic.AddInt32(&historySyncID, 1)
	fileName := fmt.Sprintf("%s/history-%d-%s-%d-%s.json",
		config.PathStorages,
		startupTime,
		cli.Store.ID.String(),
		id,
		evt.Data.SyncType.String(),
	)

	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Errorf("Failed to open file to write history sync: %v", err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err = enc.Encode(evt.Data); err != nil {
		log.Errorf("Failed to write history sync: %v", err)
		return
	}

	log.Infof("Wrote history sync to %s", fileName)
}

func handleAppState(ctx context.Context, evt *events.AppState) {
	log.Debugf("App state event: %+v / %+v", evt.Index, evt.SyncActionValue)
}
