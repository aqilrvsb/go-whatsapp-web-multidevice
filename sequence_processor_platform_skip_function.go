// Modified getDeviceWorkloads function for SequenceTriggerProcessor
// Add this to your sequence_trigger_processor.go file

// getDeviceWorkloads retrieves current device loads for balancing
// MODIFIED: Now skips devices with platform values
func (s *SequenceTriggerProcessor) getDeviceWorkloads() (map[string]DeviceLoad, error) {
	query := `
		SELECT 
			d.id,
			d.status,
			COALESCE(dlb.messages_hour, 0) as messages_hour,
			COALESCE(dlb.messages_today, 0) as messages_today,
			COALESCE(dlb.is_available, true) as is_available,
			COUNT(sc.id) as current_processing
		FROM user_devices d
		LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
		LEFT JOIN sequence_contacts sc ON sc.processing_device_id = d.id 
			AND sc.processing_started_at > NOW() - INTERVAL '5 minutes'
		WHERE d.status = 'online'
			AND (d.platform IS NULL OR d.platform = '')  -- Skip devices with platform
		GROUP BY d.id, d.status, dlb.messages_hour, dlb.messages_today, dlb.is_available
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get device workloads: %w", err)
	}
	defer rows.Close()

	loads := make(map[string]DeviceLoad)
	skippedCount := 0
	
	for rows.Next() {
		var load DeviceLoad
		if err := rows.Scan(&load.DeviceID, &load.Status, &load.MessagesHour,
			&load.MessagesToday, &load.IsAvailable, &load.CurrentProcessing); err != nil {
			continue
		}
		loads[load.DeviceID] = load
	}
	
	// Count devices with platform for logging
	var platformDevices int
	s.db.QueryRow("SELECT COUNT(*) FROM user_devices WHERE platform IS NOT NULL AND platform != ''").Scan(&platformDevices)
	
	if platformDevices > 0 {
		logrus.Debugf("Device workload query: %d devices available, %d skipped (have platform)", 
			len(loads), platformDevices)
	}

	return loads, nil
}
