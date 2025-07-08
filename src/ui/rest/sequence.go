package rest

import (
	"fmt"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/sequence"
	domainSequence "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/sequence"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type Sequence struct {
	Service sequence.ISequenceUsecase
}

func InitRestSequence(app *fiber.App, service sequence.ISequenceUsecase) {
	rest := Sequence{Service: service}
	
	// Sequence routes
	app.Get("/api/sequences", rest.GetSequences)
	app.Get("/api/sequences/summary", rest.GetSequencesSummary)
	app.Post("/api/sequences", rest.CreateSequence)
	app.Get("/api/sequences/:id", rest.GetSequenceByID)
	app.Put("/api/sequences/:id", rest.UpdateSequence)
	app.Delete("/api/sequences/:id", rest.DeleteSequence)
	
	// Contact management
	app.Post("/api/sequences/:id/contacts", rest.AddContacts)
	app.Get("/api/sequences/:id/contacts", rest.GetContacts)
	app.Delete("/api/sequences/:id/contacts/:contact_id", rest.RemoveContact)
	
	// Actions
	app.Post("/api/sequences/:id/start", rest.StartSequence)
	app.Post("/api/sequences/:id/pause", rest.PauseSequence)
	app.Post("/api/sequences/:id/toggle", rest.ToggleSequence)
	
	// UI routes
	app.Get("/sequences", rest.SequencesPage)
	app.Get("/sequences/:id", rest.SequenceDetailPage)
}

// GetSequences gets all sequences for logged in user
func (controller *Sequence) GetSequences(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Authentication required",
		})
	}
	
	sequences, err := controller.Service.GetSequences(userID)
	if err != nil {
		logrus.Errorf("Failed to get sequences for user %s: %v", userID, err)
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	logrus.Infof("Found %d sequences for user %s", len(sequences), userID)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequences retrieved",
		Results: sequences,
	})
}

// CreateSequence creates a new sequence
func (controller *Sequence) CreateSequence(c *fiber.Ctx) error {
	var request sequence.CreateSequenceRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	// Set user ID from session
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Authentication required",
		})
	}
	request.UserID = userID
	
	// Log the request
	logrus.Infof("CreateSequence request: %+v", request)
	logrus.Infof("Number of steps: %d", len(request.Steps))
	
	response, err := controller.Service.CreateSequence(request)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  201,
		Code:    "CREATED",
		Message: "Sequence created successfully",
		Results: response,
	})
}
// GetSequenceByID gets sequence details
func (controller *Sequence) GetSequenceByID(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	response, err := controller.Service.GetSequenceByID(sequenceID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Sequence not found",
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequence details retrieved",
		Results: response,
	})
}

// UpdateSequence updates a sequence
func (controller *Sequence) UpdateSequence(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	var request sequence.UpdateSequenceRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	err := controller.Service.UpdateSequence(sequenceID, request)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequence updated successfully",
	})
}

// DeleteSequence deletes a sequence
func (controller *Sequence) DeleteSequence(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	err := controller.Service.DeleteSequence(sequenceID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequence deleted successfully",
	})
}
// AddContacts adds contacts to sequence
func (controller *Sequence) AddContacts(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	var request struct {
		Contacts []string `json:"contacts"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	err := controller.Service.AddContactsToSequence(sequenceID, request.Contacts)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Added %d contacts to sequence", len(request.Contacts)),
	})
}

// GetContacts gets all contacts in sequence
func (controller *Sequence) GetContacts(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	contacts, err := controller.Service.GetSequenceContacts(sequenceID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Contacts retrieved",
		Results: contacts,
	})
}

// RemoveContact removes a contact from sequence
func (controller *Sequence) RemoveContact(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	contactID := c.Params("contact_id")
	
	err := controller.Service.RemoveContactFromSequence(sequenceID, contactID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Contact removed from sequence",
	})
}
// StartSequence starts a sequence
func (controller *Sequence) StartSequence(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	err := controller.Service.StartSequence(sequenceID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequence started",
	})
}

// PauseSequence pauses a sequence
func (controller *Sequence) PauseSequence(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	err := controller.Service.PauseSequence(sequenceID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequence paused",
	})
}

// SequencesPage renders sequences page
func (controller *Sequence) SequencesPage(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Redirect("/login")
	}
	
	// Get user's devices
	userRepo := repository.GetUserRepository()
	user, _ := userRepo.GetUserByID(userID)
	devices, _ := userRepo.GetUserDevices(user.ID)
	
	return c.Render("views/sequences", fiber.Map{
		"Title":   "Sequences",
		"User":    user,
		"Devices": devices,
	})
}

// SequenceDetailPage renders sequence detail page
func (controller *Sequence) SequenceDetailPage(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	sequence, err := controller.Service.GetSequenceByID(sequenceID)
	if err != nil {
		return c.Redirect("/sequences")
	}
	
	return c.Render("views/sequence_detail", fiber.Map{
		"Title":    "Sequence Detail",
		"Sequence": sequence,
	})
}

// GetSequencesSummary gets sequences summary for dashboard
func (controller *Sequence) GetSequencesSummary(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Authentication required",
		})
	}
	
	// Get user's devices count
	userRepo := repository.GetUserRepository()
	devices, _ := userRepo.GetUserDevices(userID)
	totalUserDevices := len(devices)
	
	sequences, err := controller.Service.GetSequences(userID)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	// Calculate summary
	summary := map[string]interface{}{
		"sequences": map[string]int{
			"total":    len(sequences),
			"active":   0,
			"inactive": 0,
		},
		"contacts": map[string]interface{}{
			"total":                0,
			"average_per_sequence": 0.0,
			"total_success":        0,
			"total_remaining":      0,
		},
		"recent_sequences": []interface{}{},
		"total_devices":    totalUserDevices,
	}
	
	// Count statuses and contacts
	totalContacts := 0
	totalSuccess := 0
	totalRemaining := 0
	
	for _, seq := range sequences {
		switch seq.Status {
		case "active":
			summary["sequences"].(map[string]int)["active"]++
		default:
			summary["sequences"].(map[string]int)["inactive"]++
		}
		totalContacts += seq.ContactsCount
		// For now, we'll just use the total contacts as remaining
		// since we don't have completed counts in SequenceResponse
		totalRemaining += seq.ContactsCount
	}
	
	// Update contact statistics
	summary["contacts"].(map[string]interface{})["total"] = totalContacts
	summary["contacts"].(map[string]interface{})["total_success"] = totalSuccess
	summary["contacts"].(map[string]interface{})["total_remaining"] = totalRemaining
	if len(sequences) > 0 {
		summary["contacts"].(map[string]interface{})["average_per_sequence"] = float64(totalContacts) / float64(len(sequences))
	}
	
	// Get recent sequences (up to 5)
	recentCount := 5
	if len(sequences) < recentCount {
		recentCount = len(sequences)
	}
	
	recentSequences := make([]interface{}, 0, recentCount)
	for i := 0; i < recentCount; i++ {
		seq := sequences[i]
		
		recentSequences = append(recentSequences, map[string]interface{}{
			"id":                  seq.ID,
			"name":                seq.Name,
			"niche":               seq.Niche,
			"status":              seq.Status,
			"contacts_count":      seq.ContactsCount,
			"total_devices":       totalUserDevices,
			"completed_contacts":  0, // TODO: Add this to SequenceResponse
			"remaining_contacts":  seq.ContactsCount,
		})
	}
	summary["recent_sequences"] = recentSequences
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Sequences summary retrieved",
		Results: summary,
	})
}

// ToggleSequence toggles sequence status between active and inactive
func (controller *Sequence) ToggleSequence(c *fiber.Ctx) error {
	sequenceID := c.Params("id")
	
	// Get current sequence
	sequence, err := controller.Service.GetSequenceByID(sequenceID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Sequence not found",
		})
	}
	
	// Toggle status
	newStatus := "inactive"
	if sequence.Status != "active" {
		newStatus = "active"
	}
	
	// Update sequence
	updateReq := domainSequence.UpdateSequenceRequest{
		Status:   newStatus,
		IsActive: newStatus == "active",
	}
	
	err = controller.Service.UpdateSequence(sequenceID, updateReq)
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Sequence %s successfully", newStatus),
		Results: map[string]string{
			"status": newStatus,
		},
	})
}