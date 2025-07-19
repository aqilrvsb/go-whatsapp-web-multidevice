// Improved enrollContactInSequence with device pre-assignment
func (s *SequenceTriggerProcessor) enrollContactInSequence(sequenceID string, lead models.Lead, trigger string) error {
    // Get current device loads for pre-assignment
    deviceLoads, err := s.getDeviceWorkloads()
    if err != nil {
        logrus.Warnf("Failed to get device workloads for pre-assignment: %v", err)
        // Continue without pre-assignment
    }
    
    // Select best device for this lead
    var assignedDeviceID string
    if lead.DeviceID != "" {
        // Prefer the device that owns the lead
        assignedDeviceID = lead.DeviceID
    } else if len(deviceLoads) > 0 {
        // Find least loaded device
        minLoad := int(^uint(0) >> 1)
        for deviceID, load := range deviceLoads {
            if load.IsAvailable && load.MessagesToday < minLoad {
                assignedDeviceID = deviceID
                minLoad = load.MessagesToday
            }
        }
    }
    
    // Get all steps for this sequence
    stepsQuery := `
        SELECT id, day_number, trigger, next_trigger, trigger_delay_hours 
        FROM sequence_steps 
        WHERE sequence_id = $1 
        ORDER BY day_number ASC
    `
    
    // ... rest of the code ...
    
    // Modified insert query with device assignment
    insertQuery := `
        INSERT INTO sequence_contacts (
            sequence_id, contact_phone, contact_name, 
            current_step, status, completed_at, current_trigger,
            next_trigger_time, sequence_stepid, assigned_device_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (sequence_id, contact_phone, sequence_stepid) DO NOTHING
    `
    
    // When inserting, include the assigned device
    _, err := s.db.Exec(insertQuery, 
        sequenceID,          // sequence_id
        lead.Phone,          // contact_phone
        lead.Name,           // contact_name
        step.DayNumber,      // current_step
        status,              // status
        currentTime,         // completed_at
        step.Trigger,        // current_trigger
        nextTriggerTime,     // next_trigger_time
        step.ID,             // sequence_stepid
        assignedDeviceID,    // assigned_device_id (NEW)
    )
}