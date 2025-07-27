package usecase

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
)

// SequenceTriggerProcessor handles trigger-based sequence processing
type SequenceTriggerProcessor struct {
	db *sql.DB
}

// NewSequenceTriggerProcessor creates new processor
func NewSequenceTriggerProcessor(db *sql.DB) *SequenceTriggerProcessor {
	return &SequenceTriggerProcessor{
		db: db,
	}
}

// ProcessSequenceTriggers is the main entry point - now uses Direct Broadcast only
func (s *SequenceTriggerProcessor) ProcessSequenceTriggers() error {
	start := time.Now()
	
	// Use Direct Broadcast processor
	directProcessor := NewDirectBroadcastProcessor(s.db)
	
	// Process enrollments
	enrolledCount, err := directProcessor.ProcessDirectEnrollments()
	if err != nil {
		logrus.Errorf("Error in direct broadcast enrollment: %v", err)
		return err
	}
	
	if enrolledCount > 0 {
		logrus.Infof("✅ Direct Broadcast: Enrolled %d leads in %v", 
			enrolledCount, time.Since(start))
	}
	
	return nil
}

// StartProcessing starts the sequence trigger processing
func (s *SequenceTriggerProcessor) StartProcessing() {
	logrus.Info("Starting Direct Broadcast Sequence Processor...")
	
	// Process immediately on startup
	if err := s.ProcessSequenceTriggers(); err != nil {
		logrus.Errorf("Initial processing error: %v", err)
	}
	
	// Then run periodically
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		if err := s.ProcessSequenceTriggers(); err != nil {
			logrus.Errorf("Processing error: %v", err)
		}
	}
}
