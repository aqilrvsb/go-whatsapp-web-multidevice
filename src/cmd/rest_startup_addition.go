	// Start auto flush chat csv
	if config.WhatsappChatStorage {
		go helpers.StartAutoFlushChatStorage()
	}
	
	// Start device health monitor
	healthMonitor := whatsapp.GetDeviceHealthMonitor(whatsappDB)
	healthMonitor.Start()
	logrus.Info("Device health monitor started")
	
	// Start broadcast manager
	_ = broadcast.GetBroadcastManager()
	logrus.Info("Broadcast manager started")
	
	// Start the optimized broadcast processor - THIS IS CRITICAL!
	// This processor polls the database for pending messages and creates workers
	go usecase.StartOptimizedBroadcastProcessor()
	logrus.Info("Optimized broadcast processor started")
	
	// Start campaign/sequence trigger processor
	go usecase.StartTriggerProcessor()
	logrus.Info("Campaign trigger processor started")
	
	// Initialize worker control API endpoints
	rest.InitWorkerControlAPI(app)
	logrus.Info("Worker control API initialized")

	if err := app.Listen(":" + config.AppPort); err != nil {
		log.Fatalln("Failed to start: ", err.Error())
	}