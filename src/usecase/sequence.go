package usecase

import (
	"fmt"
	"time"

	domainSend "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/send"
	domainSequence "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/sequence"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/models"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
)

type sequenceService struct {
	WaCli       *whatsmeow.Client
	sendService domainSend.ISendUsecase
}

func NewSequenceUsecase(waCli *whatsmeow.Client, sendService domainSend.ISendUsecase) domainSequence.ISequenceUsecase {
	return &sequenceService{
		WaCli:       waCli,
		sendService: sendService,
	}
}

// CreateSequence creates a new sequence
func (s *sequenceService) CreateSequence(request domainSequence.CreateSequenceRequest) (domainSequence.SequenceResponse, error) {
	var response domainSequence.SequenceResponse
	
	// Create sequence
	sequence := &models.Sequence{
		UserID:      request.UserID,
		DeviceID:    request.DeviceID,
		Name:        request.Name,
		Description: request.Description,
		Niche:       request.Niche,
		TotalDays:   len(request.Steps),
		IsActive:    request.IsActive,
	}
	
	repo := repository.GetSequenceRepository()
	if err := repo.CreateSequence(sequence); err != nil {
		return response, err
	}
	
	// Create steps
	for _, stepReq := range request.Steps {
		step := &models.SequenceStep{
			SequenceID:  sequence.ID,
			Day:         stepReq.Day,
			MessageType: stepReq.MessageType,
			Content:     stepReq.Content,
			MediaURL:    stepReq.MediaURL,
			Caption:     stepReq.Caption,
			SendTime:    stepReq.SendTime,
		}
		
		if err := repo.CreateSequenceStep(step); err != nil {
			logrus.Errorf("Failed to create step: %v", err)
		}
	}
	
	response = domainSequence.SequenceResponse{
		ID:          sequence.ID,
		Name:        sequence.Name,
		Description: sequence.Description,
		UserID:      sequence.UserID,
		DeviceID:    sequence.DeviceID,
		TotalSteps:  len(request.Steps),
		TotalDays:   sequence.TotalDays,
		IsActive:    sequence.IsActive,
		CreatedAt:   sequence.CreatedAt,
		UpdatedAt:   sequence.UpdatedAt,
	}
	
	return response, nil
}
// GetSequences gets all sequences for a user
func (s *sequenceService) GetSequences(userID string) ([]domainSequence.SequenceResponse, error) {
	repo := repository.GetSequenceRepository()
	sequences, err := repo.GetSequences(userID)
	if err != nil {
		return nil, err
	}
	
	var responses []domainSequence.SequenceResponse
	for _, seq := range sequences {
		// Get contact count
		contacts, _ := repo.GetSequenceContacts(seq.ID)
		
		response := domainSequence.SequenceResponse{
			ID:           seq.ID,
			Name:         seq.Name,
			Description:  seq.Description,
			UserID:       seq.UserID,
			DeviceID:     seq.DeviceID,
			TotalDays:    seq.TotalDays,
			IsActive:     seq.IsActive,
			ContactCount: len(contacts),
			CreatedAt:    seq.CreatedAt,
			UpdatedAt:    seq.UpdatedAt,
		}
		responses = append(responses, response)
	}
	
	return responses, nil
}

// GetSequenceByID gets sequence details by ID
func (s *sequenceService) GetSequenceByID(sequenceID string) (domainSequence.SequenceDetailResponse, error) {
	var response domainSequence.SequenceDetailResponse
	
	repo := repository.GetSequenceRepository()
	sequence, err := repo.GetSequenceByID(sequenceID)
	if err != nil {
		return response, err
	}
	
	// Get steps
	steps, err := repo.GetSequenceSteps(sequenceID)
	if err != nil {
		return response, err
	}
	
	// Get contacts
	contacts, _ := repo.GetSequenceContacts(sequenceID)
	
	// Get stats
	stats, _ := repo.GetSequenceStats(sequenceID)
	
	// Build response
	response.SequenceResponse = domainSequence.SequenceResponse{
		ID:           sequence.ID,
		Name:         sequence.Name,
		Description:  sequence.Description,
		UserID:       sequence.UserID,
		DeviceID:     sequence.DeviceID,
		TotalDays:    sequence.TotalDays,
		TotalSteps:   len(steps),
		IsActive:     sequence.IsActive,
		ContactCount: len(contacts),
		CreatedAt:    sequence.CreatedAt,
		UpdatedAt:    sequence.UpdatedAt,
	}
	
	// Add steps
	for _, step := range steps {
		response.Steps = append(response.Steps, domainSequence.SequenceStepResponse{
			ID:          step.ID,
			SequenceID:  step.SequenceID,
			Day:         step.Day,
			MessageType: step.MessageType,
			Content:     step.Content,
			MediaURL:    step.MediaURL,
			Caption:     step.Caption,
			SendTime:    step.SendTime,
		})
	}
	
	// Add stats
	response.Stats = domainSequence.SequenceStats{
		TotalContacts:     len(contacts),
		ActiveContacts:    stats["active"],
		CompletedContacts: stats["completed"],
		PausedContacts:    stats["paused"],
		MessagesSent:      stats["messages_sent"],
	}
	
	return response, nil
}
// UpdateSequence updates a sequence
func (s *sequenceService) UpdateSequence(sequenceID string, request domainSequence.UpdateSequenceRequest) error {
	repo := repository.GetSequenceRepository()
	
	sequence, err := repo.GetSequenceByID(sequenceID)
	if err != nil {
		return err
	}
	
	// Update fields
	if request.Name != "" {
		sequence.Name = request.Name
	}
	if request.Description != "" {
		sequence.Description = request.Description
	}
	sequence.IsActive = request.IsActive
	
	// Update sequence
	if err := repo.UpdateSequence(sequence); err != nil {
		return err
	}
	
	// Update steps if provided
	if len(request.Steps) > 0 {
		// Delete existing steps
		// TODO: Add DeleteSequenceSteps method
		
		// Create new steps
		for _, stepReq := range request.Steps {
			step := &models.SequenceStep{
				SequenceID:  sequenceID,
				Day:         stepReq.Day,
				MessageType: stepReq.MessageType,
				Content:     stepReq.Content,
				MediaURL:    stepReq.MediaURL,
				Caption:     stepReq.Caption,
				SendTime:    stepReq.SendTime,
			}
			
			if err := repo.CreateSequenceStep(step); err != nil {
				logrus.Errorf("Failed to create step: %v", err)
			}
		}
		
		// Update total days
		sequence.TotalDays = len(request.Steps)
		repo.UpdateSequence(sequence)
	}
	
	return nil
}

// DeleteSequence deletes a sequence
func (s *sequenceService) DeleteSequence(sequenceID string) error {
	repo := repository.GetSequenceRepository()
	return repo.DeleteSequence(sequenceID)
}

// AddContactsToSequence adds contacts to a sequence
func (s *sequenceService) AddContactsToSequence(sequenceID string, contacts []string) error {
	repo := repository.GetSequenceRepository()
	
	for _, phone := range contacts {
		// Sanitize phone
		whatsapp.SanitizePhone(&phone)
		
		contact := &models.SequenceContact{
			SequenceID:   sequenceID,
			ContactPhone: phone,
		}
		
		if err := repo.AddContactToSequence(contact); err != nil {
			logrus.Errorf("Failed to add contact %s: %v", phone, err)
		}
	}
	
	return nil
}
// RemoveContactFromSequence removes a contact from sequence
func (s *sequenceService) RemoveContactFromSequence(sequenceID string, contactID string) error {
	// TODO: Implement remove contact
	return nil
}

// GetSequenceContacts gets all contacts in a sequence
func (s *sequenceService) GetSequenceContacts(sequenceID string) ([]domainSequence.SequenceContactResponse, error) {
	repo := repository.GetSequenceRepository()
	contacts, err := repo.GetSequenceContacts(sequenceID)
	if err != nil {
		return nil, err
	}
	
	var responses []domainSequence.SequenceContactResponse
	for _, contact := range contacts {
		response := domainSequence.SequenceContactResponse{
			ID:            contact.ID,
			ContactPhone:  contact.ContactPhone,
			ContactName:   contact.ContactName,
			CurrentDay:    contact.CurrentDay,
			Status:        contact.Status,
			AddedAt:       contact.AddedAt,
		}
		
		if contact.LastMessageAt != nil {
			response.LastMessageAt = *contact.LastMessageAt
		}
		if contact.CompletedAt != nil {
			response.CompletedAt = contact.CompletedAt
		}
		
		responses = append(responses, response)
	}
	
	return responses, nil
}

// StartSequence starts a sequence
func (s *sequenceService) StartSequence(sequenceID string) error {
	repo := repository.GetSequenceRepository()
	sequence, err := repo.GetSequenceByID(sequenceID)
	if err != nil {
		return err
	}
	
	sequence.IsActive = true
	return repo.UpdateSequence(sequence)
}

// PauseSequence pauses a sequence
func (s *sequenceService) PauseSequence(sequenceID string) error {
	repo := repository.GetSequenceRepository()
	sequence, err := repo.GetSequenceByID(sequenceID)
	if err != nil {
		return err
	}
	
	sequence.IsActive = false
	return repo.UpdateSequence(sequence)
}
// ProcessSequences processes all active sequences (called by cron job)
func (s *sequenceService) ProcessSequences() error {
	logrus.Info("Processing sequences...")
	
	repo := repository.GetSequenceRepository()
	currentTime := time.Now()
	currentHour := fmt.Sprintf("%02d:%02d", currentTime.Hour(), currentTime.Minute())
	
	// Get all active contacts ready for next message
	contacts, err := repo.GetActiveSequenceContacts(currentTime)
	if err != nil {
		return err
	}
	
	logrus.Infof("Found %d contacts to process", len(contacts))
	
	for _, contact := range contacts {
		// Get sequence
		sequence, err := repo.GetSequenceByID(contact.SequenceID)
		if err != nil {
			continue
		}
		
		// Get next step
		steps, err := repo.GetSequenceSteps(contact.SequenceID)
		if err != nil {
			continue
		}
		
		// Find the step for current day + 1
		nextDay := contact.CurrentDay + 1
		var nextStep *models.SequenceStep
		
		for _, step := range steps {
			if step.Day == nextDay && step.SendTime <= currentHour {
				nextStep = &step
				break
			}
		}
		
		if nextStep == nil {
			continue
		}
		
		// Send message
		if err := s.sendSequenceMessage(sequence, &contact, nextStep); err != nil {
			logrus.Errorf("Failed to send message to %s: %v", contact.ContactPhone, err)
			
			// Log failure
			log := &models.SequenceLog{
				SequenceID:   contact.SequenceID,
				ContactID:    contact.ID,
				StepID:      nextStep.ID,
				Day:         nextStep.Day,
				Status:      "failed",
				ErrorMessage: err.Error(),
			}
			repo.CreateSequenceLog(log)
		} else {
			// Update contact progress
			if nextDay >= sequence.TotalDays {
				// Mark as completed
				repo.MarkContactCompleted(contact.ID)
			} else {
				// Update to next day
				repo.UpdateContactProgress(contact.ID, nextDay, "active")
			}
			
			// Log success
			log := &models.SequenceLog{
				SequenceID: contact.SequenceID,
				ContactID:  contact.ID,
				StepID:    nextStep.ID,
				Day:       nextStep.Day,
				Status:    "sent",
			}
			repo.CreateSequenceLog(log)
		}
	}
	
	return nil
}
// sendSequenceMessage sends a message for a sequence step
func (s *sequenceService) sendSequenceMessage(sequence *models.Sequence, contact *models.SequenceContact, step *models.SequenceStep) error {
	// Get the WhatsApp client for the device
	cm := whatsapp.GetClientManager()
	_, err := cm.GetClient(sequence.DeviceID)
	if err != nil {
		return fmt.Errorf("device not connected: %v", err)
	}
	
	// Use the send service with the correct client
	switch step.MessageType {
	case "text":
		request := domainSend.MessageRequest{
			Phone:   contact.ContactPhone,
			Message: step.Content,
		}
		_, err = s.sendService.SendText(nil, request)
		
	case "image":
		// For image messages, we need to handle URL
		mediaURL := step.MediaURL
		request := domainSend.ImageRequest{
			Phone:    contact.ContactPhone,
			Caption:  step.Caption,
			ImageURL: &mediaURL,
		}
		// TODO: Implement SendImageURL in send service
		_, err = s.sendService.SendImage(nil, request)
		
	case "video":
		// TODO: Implement video sending
		err = fmt.Errorf("video messages not yet implemented")
		
	case "document":
		// TODO: Implement document sending
		err = fmt.Errorf("document messages not yet implemented")
		
	default:
		err = fmt.Errorf("unknown message type: %s", step.MessageType)
	}
	
	return err
}