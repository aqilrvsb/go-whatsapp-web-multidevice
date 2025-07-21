// Sequence Trigger Processor - Reduce verbose logging

// In sequence_trigger_processor.go, change these logs from Info to Debug:

// Line ~153 - Change main processing log
logrus.Debugf("Sequence processing completed: enrolled=%d, processed=%d, devices=%d/%d, duration=%v", 
	enrolledCount, processedCount, activeDevices, totalDevices, duration)

// Line ~453 - Remove or change to Debug
// logrus.Infof("[SEQUENCE-NAME] Contact: %s, Name from sequence_contacts: '%s'", job.phone, job.name)
// DELETE THIS LINE - it was for debugging only

// Line ~483 - Change "not ready" log to Debug
logrus.Debugf("⏰ Step %d for %s not ready (triggers in %v at %v)", 
	job.currentStep, job.phone, timeRemaining, 
	job.nextTriggerTime.Format("15:04:05"))

// Line ~264 - Change enrollment log to Debug
logrus.Debugf("Enrolling contact %s in sequence %s - creating ALL %d steps", 
	lead.Phone, sequenceID, len(steps))

// Line ~280-282 - Change step creation logs to Debug
logrus.Debugf("Step 1: PENDING - will trigger at %v (NOW + 5 minutes)", 
	nextTriggerTime.Format("15:04:05"))

// Line ~291 - Change step info to Debug
logrus.Debugf("Step %d: PENDING - will activate at %v (previous + %d hours)", 
	step.DayNumber, 
	nextTriggerTime.Format("2006-01-02 15:04:05"),
	step.TriggerDelayHours)

// Line ~495 - Keep this one as Info (important)
logrus.Infof("✅ Time reached for %s step %d - processing message", 
	job.phone, job.currentStep)

// Line ~558 - Keep completion as Info (important)
logrus.Infof("📤 Successfully queued and completed step %d for %s", 
	job.currentStep, job.phone)
