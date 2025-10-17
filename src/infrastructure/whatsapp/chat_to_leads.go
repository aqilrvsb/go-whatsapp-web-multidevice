package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"time"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow/types"
)

// AutoSaveChatsToLeads automatically saves chats as leads with duplicate prevention
func AutoSaveChatsToLeads(deviceID string, userID string) error {
	logrus.Infof("=== Starting auto-save chats to leads for device: %s ===", deviceID)
	
	// Get chats from database with messages in last 6 months
	chats, err := GetChatsFromDatabase(deviceID)
	if err != nil {
		return fmt.Errorf("failed to get chats: %v", err)
	}
	
	logrus.Infof("Retrieved %d chats with conversations in last 6 months for device %s", len(chats), deviceID)
	
	// If we have chats from database, use those (they already have 6-month filter)
	if len(chats) > 0 {
		return saveChatsAsLeads(chats, deviceID, userID)
	}
	
	// If no chats in database, try to get from WhatsApp store but only those with recent activity
	cm := GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err == nil && client != nil && client.IsLoggedIn() {
		logrus.Info("No chats in database, checking WhatsApp store for recent conversations...")
		
		// Get all contacts from WhatsApp store
		contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
		if err == nil && len(contacts) > 0 {
			logrus.Infof("Found %d total contacts in WhatsApp store", len(contacts))
			
			// Filter to only contacts with recent conversations
			recentContacts := make(map[types.JID]types.ContactInfo)
			
			// Check each contact for recent messages
			for jid, contact := range contacts {
				// Skip non-user contacts
				if jid.Server != types.DefaultUserServer || jid.User == "status" {
					continue
				}
				
				// Check if this contact has recent messages in database
				hasRecentMessages, err := checkRecentMessages(deviceID, jid.String())
				if err == nil && hasRecentMessages {
					recentContacts[jid] = contact
				}
			}
			
			logrus.Infof("Filtered to %d contacts with recent conversations", len(recentContacts))
			
			if len(recentContacts) > 0 {
				return saveContactsAsLeads(recentContacts, deviceID, userID)
			}
		}
	}
	
	logrus.Info("No contacts with recent conversations found")
	return nil
}

// checkRecentMessages checks if a contact has messages in the last 6 months
func checkRecentMessages(deviceID string, chatJID string) (bool, error) {
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	var count int
	query := `
		SELECT COUNT(*) 
		FROM whatsapp_messages 
		WHERE device_id = ? 
		AND chat_jid = ?
		AND timestamp > UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 6 MONTH))
		LIMIT 1
	`
	
	err := db.QueryRow(query, deviceID, chatJID).Scan(&count)
	if err != nil {
		return false, err
	}
	
	return count > 0, nil
}

// saveContactsAsLeads saves WhatsApp contacts directly as leads
func saveContactsAsLeads(contacts map[types.JID]types.ContactInfo, deviceID string, userID string) error {
	leadRepo := repository.GetLeadRepository()
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	savedCount := 0
	skippedCount := 0
	
	for jid, contact := range contacts {
		// Skip non-user contacts (groups, broadcast, etc)
		if jid.Server != types.DefaultUserServer || jid.User == "status" {
			continue
		}
		
		phone := jid.User
		name := ""
		
		// Get best available name
		if contact.PushName != "" {
			name = contact.PushName
		} else if contact.BusinessName != "" {
			name = contact.BusinessName
		} else if contact.FullName != "" {
			name = contact.FullName
		} else if contact.FirstName != "" {
			name = contact.FirstName
		}
		
		if name == "" {
			name = phone // Use phone as name if no name available
		}
		
		// Check if lead already exists with same device_id, user_id, and phone
		var existingID string
		checkQuery := `
			SELECT id FROM leads 
			WHERE device_id = ? AND user_id = ? AND phone = ?
			LIMIT 1
		`
		err := db.QueryRow(checkQuery, deviceID, userID, phone).Scan(&existingID)
		
		if err == nil && existingID != "" {
			// Lead already exists, skip
			skippedCount++
			logrus.Debugf("Lead already exists for phone %s, skipping", phone)
			continue
		}
		
		// Create new lead
		lead := &models.Lead{
			UserID:       userID,
			DeviceID:     deviceID,
			Name:         name,
			Phone:        phone,
			Niche:        "WHATSAPP_IMPORT", // Default niche for imported contacts
			Status:       "new",
			TargetStatus: "prospect",
			Trigger:      "",
			Platform:     "whatsapp",
			Notes:        fmt.Sprintf("Auto-imported from WhatsApp on %s", time.Now().Format("2006-01-02")),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		err = leadRepo.CreateLead(lead)
		if err != nil {
			logrus.Warnf("Failed to create lead for %s: %v", phone, err)
			continue
		}
		
		savedCount++
		logrus.Debugf("Created lead for %s (%s)", name, phone)
	}
	
	logrus.Infof("=== Auto-save completed: %d saved, %d skipped (duplicates) ===", savedCount, skippedCount)
	return nil
}

// saveChatsAsLeads saves chats from database as leads (fallback method)
func saveChatsAsLeads(chats []map[string]interface{}, deviceID string, userID string) error {
	
	leadRepo := repository.GetLeadRepository()
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	savedCount := 0
	skippedCount := 0
	
	for _, chat := range chats {
		// chatJID is already in the chat map as "id"
		name := chat["name"].(string)
		phone := chat["phone"].(string)
		
		// Skip if phone is empty
		if phone == "" {
			continue
		}
		
		// Check if lead already exists with same device_id, user_id, and phone
		var existingID string
		checkQuery := `
			SELECT id FROM leads 
			WHERE device_id = ? AND user_id = ? AND phone = ?
			LIMIT 1
		`
		err := db.QueryRow(checkQuery, deviceID, userID, phone).Scan(&existingID)
		
		if err == nil && existingID != "" {
			// Lead already exists, skip
			skippedCount++
			logrus.Debugf("Lead already exists for phone %s, skipping", phone)
			continue
		}
		
		// Create new lead
		lead := &models.Lead{
			UserID:       userID,
			DeviceID:     deviceID,
			Name:         name,
			Phone:        phone,
			Niche:        "WHATSAPP_IMPORT", // Default niche for imported contacts
			Status:       "new",
			TargetStatus: "prospect",
			Trigger:      "",
			Platform:     "whatsapp",
			Notes:        fmt.Sprintf("Auto-imported from WhatsApp on %s", time.Now().Format("2006-01-02")),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		err = leadRepo.CreateLead(lead)
		if err != nil {
			logrus.Warnf("Failed to create lead for %s: %v", phone, err)
			continue
		}
		
		savedCount++
		logrus.Debugf("Created lead for %s (%s)", name, phone)
	}
	
	logrus.Infof("=== Auto-save completed: %d saved, %d skipped (duplicates) ===", savedCount, skippedCount)
	return nil
}

// PreserveExistingData ensures we never delete existing chats/messages when rescanning
func PreserveExistingData(deviceID string) {
	logrus.Infof("=== Preserving existing data for device: %s ===", deviceID)
	
	// We don't delete any existing data - just add new ones
	// The system is already designed to use UPSERT (INSERT ... ON CONFLICT)
	// for chats and messages, so existing data is preserved
}

// MergeDeviceData merges data from a new device without removing old device data
func MergeDeviceData(oldDeviceID, newDeviceID, userID string) error {
	logrus.Infof("=== Merging data from device %s to %s ===", oldDeviceID, newDeviceID)
	
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// 1. Copy chats that don't exist in new device
	copyChatsQuery := `
		INSERT INTO whatsapp_chats (device_id, chat_jid, chat_name, last_message_time, created_at)
		SELECT ?, chat_jid, chat_name, last_message_time, NOW()
		FROM whatsapp_chats
		WHERE device_id = ?
		AND chat_jid NOT IN (
			SELECT chat_jid FROM whatsapp_chats WHERE device_id = ?
		)
		ON DUPLICATE KEY UPDATE last_message_time = VALUES(last_message_time)
	`
	_, err = tx.Exec(copyChatsQuery, newDeviceID, oldDeviceID, newDeviceID)
	if err != nil {
		return fmt.Errorf("failed to copy chats: %v", err)
	}
	
	// 2. Copy messages that don't exist in new device
	copyMessagesQuery := `
		INSERT IGNORE INTO whatsapp_messages(
			device_id, chat_jid, message_id, sender_jid, 
			message_text, message_type, message_secrets, timestamp, created_at
		)
		SELECT ?, chat_jid, message_id, sender_jid,
			message_text, message_type, message_secrets, timestamp, NOW()
		FROM whatsapp_messages
		WHERE device_id = ?
		AND message_id NOT IN (
			SELECT message_id FROM whatsapp_messages WHERE device_id = ?
		)
	`
	_, err = tx.Exec(copyMessagesQuery, newDeviceID, oldDeviceID, newDeviceID)
	if err != nil {
		return fmt.Errorf("failed to copy messages: %v", err)
	}
	
	// 3. Copy leads that don't exist for new device
	copyLeadsQuery := `
		INSERT INTO leads(
			user_id, device_id, name, phone, niche, 
			status, target_status, ` + "`trigger`" + `, journey, created_at, updated_at
		)
		SELECT user_id, ?, name, phone, niche,
			status, target_status, ` + "`trigger`" + `, 
			CONCAT(COALESCE(journey, ''), '\n[Copied FROM device: ', ?, ']'),
			NOW(), NOW()
		FROM leads
		WHERE device_id = ? AND user_id = ?
		AND phone NOT IN (
			SELECT phone FROM leads WHERE device_id = ? AND user_id = ?
		)
	`
	_, err = tx.Exec(copyLeadsQuery, newDeviceID, oldDeviceID, oldDeviceID, userID, newDeviceID, userID)
	if err != nil {
		return fmt.Errorf("failed to copy leads: %v", err)
	}
	
	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	
	logrus.Info("=== Successfully merged device data ===")
	return nil
}

// HandleDeviceChange handles when a device is banned and replaced with new number
func HandleDeviceChange(oldDeviceID, newDeviceID, userID string) error {
	logrus.Infof("=== Handling device change from %s to %s ===", oldDeviceID, newDeviceID)
	
	// 1. First preserve all existing data
	PreserveExistingData(oldDeviceID)
	
	// 2. Merge data from old device to new device
	err := MergeDeviceData(oldDeviceID, newDeviceID, userID)
	if err != nil {
		return fmt.Errorf("failed to merge device data: %v", err)
	}
	
	// 3. Auto-save new chats to leads
	err = AutoSaveChatsToLeads(newDeviceID, userID)
	if err != nil {
		logrus.Warnf("Failed to auto-save chats to leads: %v", err)
	}
	
	return nil
}

// ExtractPhoneFromJID extracts clean phone number from WhatsApp JID
func ExtractPhoneFromJID(jid string) string {
	// Remove @s.whatsapp.net suffix
	phone := strings.TrimSuffix(jid, "@s.whatsapp.net")
	
	// Parse JID if needed
	if strings.Contains(phone, "@") {
		parsedJID, err := types.ParseJID(jid)
		if err == nil {
			phone = parsedJID.User
		}
	}
	
	return phone
}
