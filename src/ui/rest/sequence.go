package rest

import (
	"fmt"
	
	"github.com/aldinokemal/go-whatsapp-web-multidevice/domains/sequence"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/gofiber/fiber/v2"
)

type Sequence struct {
	Service sequence.ISequenceUsecase
}

func InitRestSequence(app *fiber.App, service sequence.ISequenceUsecase) {
	rest := Sequence{Service: service}
	
	// Sequence routes
	app.Get("/api/sequences", rest.GetSequences)
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
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	
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