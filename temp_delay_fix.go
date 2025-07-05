// StartMultiDeviceAutoReconnect starts the auto-reconnect process with proper delays
func StartMultiDeviceAutoReconnect() {
	go func() {
		// Wait longer for all services to initialize properly
		// This includes database connections, WhatsApp store, etc.
		logrus.Info("Waiting 30 seconds for all services to initialize before auto-reconnect...")
		time.Sleep(30 * time.Second)
		
		// Run initial reconnect
		MultiDeviceAutoReconnect()
		
		// Run periodic checks every 30 minutes (not too frequent for 3000 devices)
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			logrus.Info("Running periodic multi-device reconnect check...")
			MultiDeviceAutoReconnect()
		}
	}()
	
	logrus.Info("Multi-device auto-reconnect scheduled (30s delay, 30min intervals)")
}