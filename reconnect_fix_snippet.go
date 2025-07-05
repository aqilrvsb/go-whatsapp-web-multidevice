// reconnectDevice attempts to reconnect a single device using DeviceManager
func reconnectDevice(deviceID, userID, deviceName, jid, phone string) bool {
	logrus.Infof("Checking device %s (%s) - JID: %s", deviceName, deviceID, jid)
	
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Panic recovered while reconnecting device %s: %v", deviceName, r)
		}
	}()
	
	// Get DeviceManager instance
	dm := multidevice.GetDeviceManager()
	if dm == nil {
		logrus.Errorf("DeviceManager is nil, cannot reconnect device %s", deviceName)
		return false
	}
	
	// First try to get existing connection
	conn, err := dm.GetDeviceConnection(deviceID)
	if err != nil {
		// No connection in memory, try to create one
		logrus.Infof("No connection in memory for device %s, creating new connection...", deviceName)
		
		// Add validation
		if userID == "" {
			logrus.Errorf("Empty userID for device %s, cannot create connection", deviceName)
			return false
		}
		
		conn, err = dm.GetOrCreateDeviceConnection(deviceID, userID, phone)
		if err != nil {
			logrus.Errorf("Failed to create connection for device %s: %v", deviceName, err)
			return false
		}
	}
	
	// Rest of the function remains the same...
}