package rest

import (
	"fmt"

	domainGroup "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/group"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/middleware"
	"github.com/gofiber/fiber/v2"
	"go.mau.fi/whatsmeow"
)

type Group struct {
	Service domainGroup.IGroupUsecase
}

func InitRestGroup(app *fiber.App, service domainGroup.IGroupUsecase) Group {
	rest := Group{Service: service}
	
	// Apply CustomAuth middleware to all group endpoints
	groupRoutes := app.Group("/group", middleware.CustomAuth())
	
	groupRoutes.Post("/", rest.CreateGroup)
	groupRoutes.Post("/join-with-link", rest.JoinGroupWithLink)
	groupRoutes.Post("/leave", rest.LeaveGroup)
	groupRoutes.Post("/participants", rest.AddParticipants)
	groupRoutes.Post("/participants/remove", rest.DeleteParticipants)
	groupRoutes.Post("/participants/promote", rest.PromoteParticipants)
	groupRoutes.Post("/participants/demote", rest.DemoteParticipants)
	groupRoutes.Get("/participant-requests", rest.ListParticipantRequests)
	groupRoutes.Post("/participant-requests/approve", rest.ApproveParticipantRequests)
	groupRoutes.Post("/participant-requests/reject", rest.RejectParticipantRequests)
	
	return rest
}

func (controller *Group) JoinGroupWithLink(c *fiber.Ctx) error {
	var request domainGroup.JoinGroupWithLinkRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	// Get user context from authentication
	userID, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User authentication required",
		})
	}
	
	// Validate device ownership if device_id is provided
	if request.DeviceID != "" {
		if err := validateDeviceOwnership(c, userID, request.DeviceID); err != nil {
			return err
		}
	}

	response, err := controller.Service.JoinGroupWithLink(c.UserContext(), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success joined group",
		Results: map[string]string{
			"group_id": response,
		},
	})
}

func (controller *Group) LeaveGroup(c *fiber.Ctx) error {
	var request domainGroup.LeaveGroupRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	// Get user context from authentication
	userID, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User authentication required",
		})
	}
	
	// Validate device ownership
	if request.DeviceID != "" {
		if err := validateDeviceOwnership(c, userID, request.DeviceID); err != nil {
			return err
		}
	}

	whatsapp.SanitizePhone(&request.GroupID)

	err = controller.Service.LeaveGroup(c.UserContext(), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success leave group",
	})
}

func (controller *Group) CreateGroup(c *fiber.Ctx) error {
	var request domainGroup.CreateGroupRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)

	// Get user context from authentication
	userID, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User authentication required",
		})
	}
	
	// Validate device ownership
	if request.DeviceID != "" {
		if err := validateDeviceOwnership(c, userID, request.DeviceID); err != nil {
			return err
		}
	}

	groupID, err := controller.Service.CreateGroup(c.UserContext(), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Success created group with id %s", groupID),
		Results: map[string]string{
			"group_id": groupID,
		},
	})
}

func (controller *Group) AddParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangeAdd, "Success add participants")
}

func (controller *Group) DeleteParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangeRemove, "Success delete participants")
}

func (controller *Group) PromoteParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangePromote, "Success promote participants")
}

func (controller *Group) DemoteParticipants(c *fiber.Ctx) error {
	return controller.manageParticipants(c, whatsmeow.ParticipantChangeDemote, "Success demote participants")
}

func (controller *Group) ListParticipantRequests(c *fiber.Ctx) error {
	var request domainGroup.GetGroupRequestParticipantsRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)

	if request.GroupID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_GROUP_ID",
			Message: "Group ID cannot be empty",
		})
	}

	// Get user context from authentication
	userID, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User authentication required",
		})
	}
	
	// Validate device ownership
	if request.DeviceID != "" {
		if err := validateDeviceOwnership(c, userID, request.DeviceID); err != nil {
			return err
		}
	}

	whatsapp.SanitizePhone(&request.GroupID)

	result, err := controller.Service.GetGroupRequestParticipants(c.UserContext(), request)
	utils.PanicIfNeeded(err)

	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Success getting list requested participants",
		Results: result,
	})
}

func (controller *Group) ApproveParticipantRequests(c *fiber.Ctx) error {
	return controller.handleRequestedParticipants(c, whatsmeow.ParticipantChangeApprove, "Success approve requested participants")
}

func (controller *Group) RejectParticipantRequests(c *fiber.Ctx) error {
	return controller.handleRequestedParticipants(c, whatsmeow.ParticipantChangeReject, "Success reject requested participants")
}

// Generalized participant management handler
func (controller *Group) manageParticipants(c *fiber.Ctx, action whatsmeow.ParticipantChange, successMsg string) error {
	var request domainGroup.ParticipantRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	
	// Get user context from authentication
	userID, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User authentication required",
		})
	}
	
	// Validate device ownership
	if request.DeviceID != "" {
		if err := validateDeviceOwnership(c, userID, request.DeviceID); err != nil {
			return err
		}
	}
	
	whatsapp.SanitizePhone(&request.GroupID)
	request.Action = action
	result, err := controller.Service.ManageParticipant(c.UserContext(), request)
	utils.PanicIfNeeded(err)
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: successMsg,
		Results: result,
	})
}

// Generalized requested participants handler
func (controller *Group) handleRequestedParticipants(c *fiber.Ctx, action whatsmeow.ParticipantRequestChange, successMsg string) error {
	var request domainGroup.GroupRequestParticipantsRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	
	// Get user context from authentication
	userID, ok := middleware.GetUserFromContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "User authentication required",
		})
	}
	
	// Validate device ownership
	if request.DeviceID != "" {
		if err := validateDeviceOwnership(c, userID, request.DeviceID); err != nil {
			return err
		}
	}
	
	whatsapp.SanitizePhone(&request.GroupID)
	request.Action = action
	result, err := controller.Service.ManageGroupRequestParticipants(c.UserContext(), request)
	utils.PanicIfNeeded(err)
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: successMsg,
		Results: result,
	})
}
