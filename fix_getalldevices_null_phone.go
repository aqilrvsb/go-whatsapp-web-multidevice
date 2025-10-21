package repository

import (
    "database/sql"
    "fmt"
)

// Fix for GetAllDevices function to handle NULL phone values
// Replace the existing GetAllDevices function in user_repository.go with this:

func (r *UserRepository) GetAllDevices() ([]*models.UserDevice, error) {
    query := `
        SELECT id, user_id, device_name, 
               COALESCE(phone, '') as phone,  -- Handle NULL phone values
               status, last_seen, created_at, updated_at,
               COALESCE(jid, '') as jid, 
               COALESCE(platform, '') as platform
        FROM user_devices 
        ORDER BY created_at DESC
    `
    
    rows, err := r.db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("failed to query all devices: %w", err)
    }
    defer rows.Close()
    
    var devices []*models.UserDevice
    for rows.Next() {
        device := &models.UserDevice{}
        err := rows.Scan(
            &device.ID,
            &device.UserID,
            &device.DeviceName,
            &device.Phone,        // Now this will never be NULL
            &device.Status,
            &device.LastSeen,
            &device.CreatedAt,
            &device.UpdatedAt,
            &device.JID,
            &device.Platform,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan device row: %w", err)
        }
        
        // Platform devices always appear as online
        if device.Platform != "" {
            device.Status = "online"
        }
        
        devices = append(devices, device)
    }
    
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating device rows: %w", err)
    }
    
    return devices, nil
}
