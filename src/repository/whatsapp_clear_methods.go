package repository

import (
	"fmt"
)

// ClearDeviceMessages clears all messages for a device
func (r *WhatsAppRepository) ClearDeviceMessages(deviceID string) error {
	query := `DELETE FROM whatsapp_messages WHERE device_id = $1`
	_, err := r.db.Exec(query, deviceID)
	if err != nil {
		return fmt.Errorf("failed to clear device messages: %w", err)
	}
	return nil
}

// ClearDeviceChats clears all chats for a device
func (r *WhatsAppRepository) ClearDeviceChats(deviceID string) error {
	query := `DELETE FROM whatsapp_chats WHERE device_id = $1`
	_, err := r.db.Exec(query, deviceID)
	if err != nil {
		return fmt.Errorf("failed to clear device chats: %w", err)
	}
	return nil
}