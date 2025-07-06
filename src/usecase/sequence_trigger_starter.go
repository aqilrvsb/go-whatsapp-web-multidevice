package usecase

import (
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

// Global sequence trigger processor instance
var sequenceTriggerProcessor *SequenceTriggerProcessor

// StartSequenceTriggerProcessor initializes and starts the sequence trigger processor
func StartSequenceTriggerProcessor() {
	logrus.Info("Initializing sequence trigger processor...")
	
	// Get database connection
	userRepo := repository.GetUserRepository()
	db := userRepo.DB()
	
	// Create processor
	sequenceTriggerProcessor = NewSequenceTriggerProcessor(db)
	
	// Start processing
	if err := sequenceTriggerProcessor.Start(); err != nil {
		logrus.Errorf("Failed to start sequence trigger processor: %v", err)
		return
	}
	
	logrus.Info("Sequence trigger processor started successfully")
}

// StopSequenceTriggerProcessor stops the sequence trigger processor
func StopSequenceTriggerProcessor() {
	if sequenceTriggerProcessor != nil {
		sequenceTriggerProcessor.Stop()
		logrus.Info("Sequence trigger processor stopped")
	}
}

// GetSequenceTriggerProcessor returns the global instance
func GetSequenceTriggerProcessor() *SequenceTriggerProcessor {
	return sequenceTriggerProcessor
}