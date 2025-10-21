package usecase

import (
	"database/sql"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	sequenceProcessorMutex sync.Mutex
	lastSequenceProcess    time.Time
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

// ProcessSequenceTriggers is the main entry point - now processes BOTH sequences and campaigns
func (s *SequenceTriggerProcessor) ProcessSequenceTriggers() error {
	// CRITICAL FIX: Prevent concurrent processing
	if !sequenceProcessorMutex.TryLock() {
		logrus.Debug("Sequence processor already running, skipping this run")
		return nil
	}
	defer sequenceProcessorMutex.Unlock()
	
	// Also prevent processing too frequently
	if time.Since(lastSequenceProcess) < 4*time.Minute {
		logrus.Debug("Sequence processor ran recently, skipping")
		return nil
	}
	lastSequenceProcess = time.Now()
	
	start := time.Now()
	
	// Use Direct Broadcast processor
	directProcessor := NewDirectBroadcastProcessor(s.db)
	
	// Process sequence enrollments
	enrolledCount, err := directProcessor.ProcessDirectEnrollments()
	if err != nil {
		logrus.Errorf("Error in direct broadcast enrollment: %v", err)
		return err
	}
	
	if enrolledCount > 0 {
		logrus.Infof("✅ Sequences: Enrolled %d leads in %v", 
			enrolledCount, time.Since(start))
	}
	
	// Process campaigns too (NEW)
	campaignCount, err := directProcessor.ProcessCampaigns()
	if err != nil {
		logrus.Errorf("Error in campaign processing: %v", err)
		// Don't return error - continue even if campaigns fail
	}
	
	if campaignCount > 0 {
		logrus.Infof("✅ Campaigns: Processed %d campaigns", campaignCount)
	}
	
	return nil
}

// StartProcessing starts the sequence trigger processing
func (s *SequenceTriggerProcessor) StartProcessing() {
	logrus.Info("Starting Direct Broadcast Processor (Sequences + Campaigns)...")
	
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
