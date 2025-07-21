package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// AddGroupParticipants adds participants to an existing group
func AddGroupParticipants(c *fiber.Ctx) error {
	// Parse request
	var request struct {
		DeviceID     string   `json:"device_id"`
		GroupID      string   `json:"group_id"`
		Participants []string `json:"participants"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	// Validate session
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	// If device_id not provided, get first connected device
	deviceID := request.DeviceID
	if deviceID == "" {
		devices, err := userRepo.GetUserDevices(session.UserID)
		if err != nil {
			return c.Status(500).JSON(utils.ResponseData{
				Status:  500,
				Code:    "ERROR",
				Message: "Failed to get devices",
			})
		}
		
		// Find first connected device
		cm := whatsapp.GetClientManager()
		for _, device := range devices {
			if device.Platform != "" {
				continue // Skip platform devices
			}
			
			client, err := cm.GetClient(device.ID)
			if err == nil && client != nil && client.IsConnected() {
				deviceID = device.ID
				break
			}
		}
		
		if deviceID == "" {
			return c.Status(400).JSON(utils.ResponseData{
				Status:  400,
				Code:    "NO_DEVICE",
				Message: "No connected device found",
			})
		}
	}
	
	// Get WhatsApp client
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not connected",
		})
	}
	
	// Parse group JID
	groupJID, err := types.ParseJID(request.GroupID)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_JID",
			Message: "Invalid group ID",
		})
	}
	
	// Convert phone numbers to JIDs
	var participantJIDs []types.JID
	for _, phone := range request.Participants {
		// Clean phone number - remove any non-digits
		cleanPhone := ""
		for _, ch := range phone {
			if ch >= '0' && ch <= '9' {
				cleanPhone += string(ch)
			}
		}
		
		if cleanPhone != "" {
			jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
			if err == nil {
				participantJIDs = append(participantJIDs, jid)
			}
		}
	}
	
	if len(participantJIDs) == 0 {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "No valid participants",
		})
	}
	
	// Add participants to group
	resp, err := client.UpdateGroupParticipants(groupJID, participantJIDs, whatsmeow.ParticipantChangeAdd)
	if err != nil {
		logrus.Errorf("Failed to add participants to group: %v", err)
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to add participants to group",
		})
	}
	
	// Process response
	results := make([]map[string]interface{}, 0)
	for jid, result := range resp {
		status := "success"
		message := "Added to group successfully"
		
		// Check if failed
		if result.Error != 0 {
			status = "failed"
			switch result.Error {
			case 403:
				message = "Not authorized"
			case 404:
				message = "User not found"
			case 409:
				message = "Already in group"
			case 500:
				message = "Server error"
			default:
				if result.AddRequest != nil {
					message = "Invite sent"
					status = "pending"
				} else {
					message = "Failed to add"
				}
			}
		}
		
		// Get phone number from JID
		phone := jid
		
		results = append(results, map[string]interface{}{
			"participant": phone,
			"status":      status,
			"message":     message,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Participants processed",
		Results: results,
	})
}

// AddCommunityParticipants adds participants to a community
func AddCommunityParticipants(c *fiber.Ctx) error {
	// Parse request
	var request struct {
		DeviceID     string   `json:"device_id"`
		CommunityID  string   `json:"community_id"`
		Participants []string `json:"participants"`
	}
	
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
	// Validate session
	sessionToken := c.Cookies("session_token")
	if sessionToken == "" {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "No session token",
		})
	}
	
	userRepo := repository.GetUserRepository()
	session, err := userRepo.GetSession(sessionToken)
	if err != nil {
		return c.Status(401).JSON(utils.ResponseData{
			Status:  401,
			Code:    "UNAUTHORIZED",
			Message: "Invalid session",
		})
	}
	
	// If device_id not provided, get first connected device
	deviceID := request.DeviceID
	if deviceID == "" {
		devices, err := userRepo.GetUserDevices(session.UserID)
		if err != nil {
			return c.Status(500).JSON(utils.ResponseData{
				Status:  500,
				Code:    "ERROR",
				Message: "Failed to get devices",
			})
		}
		
		// Find first connected device
		cm := whatsapp.GetClientManager()
		for _, device := range devices {
			if device.Platform != "" {
				continue // Skip platform devices
			}
			
			client, err := cm.GetClient(device.ID)
			if err == nil && client != nil && client.IsConnected() {
				deviceID = device.ID
				break
			}
		}
		
		if deviceID == "" {
			return c.Status(400).JSON(utils.ResponseData{
				Status:  400,
				Code:    "NO_DEVICE",
				Message: "No connected device found",
			})
		}
	}
	
	// Get WhatsApp client
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Device not connected",
		})
	}
	
	// Parse community JID
	communityJID, err := types.ParseJID(request.CommunityID)
	if err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "INVALID_JID",
			Message: "Invalid community ID",
		})
	}
	
	// Get community info to find announcement group
	groups, err := client.GetJoinedGroups()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to get groups",
		})
	}
	
	// Find the announcement group for this community
	var announcementGroupJID types.JID
	for _, group := range groups {
		// Check if this is the community's announcement group
		if group.IsParent && group.JID.String() == communityJID.String() {
			// For communities, we need to add to a linked group, not the parent
			// Try to get sub-groups
			subGroups, err := client.GetSubGroups(communityJID)
			if err == nil && len(subGroups) > 0 {
				// Use the first sub-group (usually announcement group)
				announcementGroupJID = subGroups[0].JID
				break
			}
		}
		
		// Alternative: check if it's linked to the community
		if !group.LinkedParentJID.IsEmpty() && group.LinkedParentJID.String() == communityJID.String() {
			// This is a group linked to our community
			if group.IsAnnounce {
				// Prefer announcement groups
				announcementGroupJID = group.JID
				break
			} else if announcementGroupJID.IsEmpty() {
				// Use any linked group if no announcement group found
				announcementGroupJID = group.JID
			}
		}
	}
	
	if announcementGroupJID.IsEmpty() {
		return c.Status(404).JSON(utils.ResponseData{
			Status:  404,
			Code:    "NOT_FOUND",
			Message: "Community announcement group not found",
		})
	}
	
	// Convert phone numbers to JIDs
	var participantJIDs []types.JID
	for _, phone := range request.Participants {
		// Clean phone number - remove any non-digits
		cleanPhone := ""
		for _, ch := range phone {
			if ch >= '0' && ch <= '9' {
				cleanPhone += string(ch)
			}
		}
		
		if cleanPhone != "" {
			jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
			if err == nil {
				participantJIDs = append(participantJIDs, jid)
			}
		}
	}
	
	if len(participantJIDs) == 0 {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "No valid participants",
		})
	}
	
	// Add participants to announcement group
	resp, err := client.UpdateGroupParticipants(announcementGroupJID, participantJIDs, whatsmeow.ParticipantChangeAdd)
	if err != nil {
		logrus.Errorf("Failed to add participants to community: %v", err)
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: "Failed to add participants to community",
		})
	}
	
	// Process response
	results := make([]map[string]interface{}, 0)
	for jid, result := range resp {
		status := "success"
		message := "Added to community successfully"
		
		// Check if failed
		if result.Error != 0 {
			status = "failed"
			switch result.Error {
			case 403:
				message = "Not authorized"
			case 404:
				message = "User not found"
			case 409:
				message = "Already in community"
			case 500:
				message = "Server error"
			default:
				if result.AddRequest != nil {
					message = "Invite sent"
					status = "pending"
				} else {
					message = "Failed to add"
				}
			}
		}
		
		// Get phone number from JID
		phone := jid
		
		results = append(results, map[string]interface{}{
			"participant": phone,
			"status":      status,
			"message":     message,
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Participants processed",
		Results: results,
	})
}
