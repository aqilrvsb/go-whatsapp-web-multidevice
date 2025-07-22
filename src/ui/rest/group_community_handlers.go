package rest

import (
	"fmt"
	"strings"
	
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
	phoneMap := make(map[string]string) // Clean phone -> original phone	
	for _, phone := range request.Participants {
		// Clean phone number - remove any non-digits except +
		cleanPhone := ""
		for _, ch := range phone {
			if (ch >= '0' && ch <= '9') || ch == '+' {
				cleanPhone += string(ch)
			}
		}
		
		// Remove leading + if present
		cleanPhone = strings.TrimPrefix(cleanPhone, "+")
		
		if cleanPhone != "" {
			jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
			if err == nil {
				participantJIDs = append(participantJIDs, jid)
				phoneMap[cleanPhone] = phone
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
			Message: fmt.Sprintf("Failed to add participants: %v", err),
		})
	}
	
	// Process response - resp is []types.GroupParticipant
	results := make([]map[string]interface{}, 0)
	
	// Create a map of JIDs we tried to add
	attemptedJIDs := make(map[string]bool)
	for _, jid := range participantJIDs {
		attemptedJIDs[jid.User] = true
	}
	
	// Process each participant in response
	for _, participant := range resp {
		phone := participant.JID.User
		originalPhone := phoneMap[phone]
		if originalPhone == "" {
			originalPhone = phone
		}
		
		status := "success"
		message := "Added to group successfully"
		
		// Check error status
		if participant.Error != 0 {
			status = "failed"
			switch participant.Error {
			case 403:
				message = "Not authorized"
			case 404:
				message = "User not found on WhatsApp"
			case 409:
				message = "Already in group"
			case 500:
				message = "Server error"
			default:
				message = fmt.Sprintf("Error code: %d", participant.Error)
			}
		} else if participant.AddRequest != nil {
			status = "pending"
			message = "Invite sent"
		}
		
		results = append(results, map[string]interface{}{
			"participant": originalPhone,
			"status":      status,
			"message":     message,
		})
		
		// Mark as processed
		delete(attemptedJIDs, phone)
	}
	
	// Add failed participants that weren't in response
	for phone := range attemptedJIDs {
		originalPhone := phoneMap[phone]
		if originalPhone == "" {
			originalPhone = phone
		}
		results = append(results, map[string]interface{}{
			"participant": originalPhone,
			"status":      "failed",
			"message":     "No response from server",
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
	// For communities, we need to find a group to add participants to
	// Communities themselves cannot have participants added directly
	groups, err := client.GetJoinedGroups()
	if err != nil {
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR", 
			Message: "Failed to get groups",
		})
	}
	
	// Find a suitable group linked to this community
	var targetGroupJID types.JID
	for _, group := range groups {
		// Check if this group is linked to the community
		if !group.LinkedParentJID.IsEmpty() && group.LinkedParentJID.String() == communityJID.String() {
			// Prefer general groups over announcement groups
			if !group.IsAnnounce {
				targetGroupJID = group.JID
				break
			} else if targetGroupJID.IsEmpty() {
				targetGroupJID = group.JID
			}
		}
	}
	
	if targetGroupJID.IsEmpty() {
		// Try to get sub-groups if we have the community
		subGroups, err := client.GetSubGroups(communityJID)
		if err == nil && len(subGroups) > 0 {
			targetGroupJID = subGroups[0].JID
		} else {
			return c.Status(404).JSON(utils.ResponseData{
				Status:  404,
				Code:    "NOT_FOUND",
				Message: "No suitable group found for this community. Please ensure the community has at least one group.",
			})
		}
	}	
	// Convert phone numbers to JIDs
	var participantJIDs []types.JID
	phoneMap := make(map[string]string) // Clean phone -> original phone
	
	for _, phone := range request.Participants {
		// Clean phone number - remove any non-digits except +
		cleanPhone := ""
		for _, ch := range phone {
			if (ch >= '0' && ch <= '9') || ch == '+' {
				cleanPhone += string(ch)
			}
		}
		
		// Remove leading + if present
		cleanPhone = strings.TrimPrefix(cleanPhone, "+")
		
		if cleanPhone != "" {
			jid, err := types.ParseJID(cleanPhone + "@s.whatsapp.net")
			if err == nil {
				participantJIDs = append(participantJIDs, jid)
				phoneMap[cleanPhone] = phone
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
	
	// Add participants to the target group
	resp, err := client.UpdateGroupParticipants(targetGroupJID, participantJIDs, whatsmeow.ParticipantChangeAdd)
	if err != nil {
		logrus.Errorf("Failed to add participants to community group: %v", err)
		return c.Status(500).JSON(utils.ResponseData{
			Status:  500,
			Code:    "ERROR",
			Message: fmt.Sprintf("Failed to add participants: %v", err),
		})
	}	
	// Process response - resp is []types.GroupParticipant
	results := make([]map[string]interface{}, 0)
	
	// Create a map of JIDs we tried to add
	attemptedJIDs := make(map[string]bool)
	for _, jid := range participantJIDs {
		attemptedJIDs[jid.User] = true
	}
	
	// Process each participant in response
	for _, participant := range resp {
		phone := participant.JID.User
		originalPhone := phoneMap[phone]
		if originalPhone == "" {
			originalPhone = phone
		}
		
		status := "success"
		message := "Added to community successfully"
		
		// Check error status
		if participant.Error != 0 {
			status = "failed"
			switch participant.Error {
			case 403:
				message = "Not authorized"
			case 404:
				message = "User not found on WhatsApp"
			case 409:
				message = "Already in community"
			case 500:
				message = "Server error"
			default:
				message = fmt.Sprintf("Error code: %d", participant.Error)
			}
		} else if participant.AddRequest != nil {
			status = "pending"
			message = "Invite sent"
		}
		
		results = append(results, map[string]interface{}{
			"participant": originalPhone,
			"status":      status,
			"message":     message,
		})
		
		// Mark as processed
		delete(attemptedJIDs, phone)
	}
	
	// Add failed participants that weren't in response
	for phone := range attemptedJIDs {
		originalPhone := phoneMap[phone]
		if originalPhone == "" {
			originalPhone = phone
		}
		results = append(results, map[string]interface{}{
			"participant": originalPhone,
			"status":      "failed",
			"message":     "No response from server",
		})
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Participants processed",
		Results: results,
	})
}