package rest

import (
	"fmt"
	
	domainCommunity "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/community"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/gofiber/fiber/v2"
)

type Community struct {
	Service domainCommunity.ICommunityUsecase
}

func InitRestCommunity(app *fiber.App, service domainCommunity.ICommunityUsecase) Community {
	rest := Community{Service: service}
	
	// Community management endpoints
	app.Post("/community", rest.CreateCommunity)
	app.Get("/community", rest.GetCommunityInfo)
	app.Post("/community/participants", rest.AddParticipants)
	app.Post("/community/link-group", rest.LinkGroup)
	app.Post("/community/unlink-group", rest.UnlinkGroup)
	
	return rest
}

// CreateCommunity creates a new WhatsApp community
func (controller *Community) CreateCommunity(c *fiber.Ctx) error {
	var request domainCommunity.CreateCommunityRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	
	communityID, err := controller.Service.CreateCommunity(c.UserContext(), request)
	utils.PanicIfNeeded(err)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Successfully created community with ID %s", communityID),
		Results: map[string]string{
			"community_id": communityID,
		},
	})
}
// GetCommunityInfo retrieves information about a community
func (controller *Community) GetCommunityInfo(c *fiber.Ctx) error {
	var request domainCommunity.GetCommunityInfoRequest
	err := c.QueryParser(&request)
	utils.PanicIfNeeded(err)
	
	if request.CommunityID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_COMMUNITY_ID",
			Message: "Community ID cannot be empty",
		})
	}
	
	whatsapp.SanitizePhone(&request.CommunityID)
	
	info, err := controller.Service.GetCommunityInfo(c.UserContext(), request)
	utils.PanicIfNeeded(err)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Successfully retrieved community info",
		Results: info,
	})
}

// AddParticipants adds participants to a community
func (controller *Community) AddParticipants(c *fiber.Ctx) error {
	var request domainCommunity.AddParticipantsRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	
	whatsapp.SanitizePhone(&request.CommunityID)
	for i := range request.Participants {
		whatsapp.SanitizePhone(&request.Participants[i])
	}
	
	result, err := controller.Service.AddParticipantsToCommunity(c.UserContext(), request)
	utils.PanicIfNeeded(err)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Participants processed",
		Results: result,
	})
}

// LinkGroup links an existing group to a community
func (controller *Community) LinkGroup(c *fiber.Ctx) error {
	var request domainCommunity.LinkGroupRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	
	whatsapp.SanitizePhone(&request.CommunityID)
	whatsapp.SanitizePhone(&request.GroupID)
	
	err = controller.Service.LinkGroupToCommunity(c.UserContext(), request)
	utils.PanicIfNeeded(err)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Group successfully linked to community",
	})
}

// UnlinkGroup unlinks a group from a community
func (controller *Community) UnlinkGroup(c *fiber.Ctx) error {
	var request domainCommunity.UnlinkGroupRequest
	err := c.BodyParser(&request)
	utils.PanicIfNeeded(err)
	
	whatsapp.SanitizePhone(&request.GroupID)
	
	err = controller.Service.UnlinkGroupFromCommunity(c.UserContext(), request)
	utils.PanicIfNeeded(err)
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Group successfully unlinked from community",
	})
}
