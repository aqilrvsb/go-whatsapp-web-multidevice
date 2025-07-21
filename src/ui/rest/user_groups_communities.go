package rest

import (
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/sirupsen/logrus"
)

type GroupInfo struct {
	JID         string `json:"JID"`
	Name        string `json:"Name"`
	IsAdmin     bool   `json:"IsAdmin"`
	Participants int   `json:"Participants"`
}

type CommunityInfo struct {
	JID         string `json:"JID"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	IsAdmin     bool   `json:"IsAdmin"`
}

// GetUserGroups gets all groups for the authenticated user
func GetUserGroups(c *fiber.Ctx) error {
	// Get user from session
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
	
	// Get user's primary device
	devices, err := userRepo.GetUserDevices(session.UserID)
	if err != nil || len(devices) == 0 {
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "No devices found",
			Results: map[string]interface{}{
				"data": []GroupInfo{},
			},
		})
	}
	
	// Use the first device
	deviceID := devices[0].ID
	
	// Get WhatsApp client
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Device not connected",
			Results: map[string]interface{}{
				"data": []GroupInfo{},
			},
		})
	}
	
	// Get all groups
	groups, err := client.GetJoinedGroups()
	if err != nil {
		logrus.Errorf("Failed to get groups: %v", err)
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Failed to get groups",
			Results: map[string]interface{}{
				"data": []GroupInfo{},
			},
		})
	}
	
	// Convert to our format - filter out communities (parent groups)
	groupList := make([]GroupInfo, 0, len(groups))
	for _, group := range groups {
		// Skip communities (parent groups) - only include regular groups
		if group.IsParent {
			continue
		}
		
		groupInfo := GroupInfo{
			JID:         group.JID.String(),
			Name:        group.GroupName.Name,
			Participants: len(group.Participants),
		}
		
		// Check if user is admin
		for _, participant := range group.Participants {
			if participant.JID.User == client.Store.ID.User && participant.IsAdmin {
				groupInfo.IsAdmin = true
				break
			}
		}
		
		groupList = append(groupList, groupInfo)
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Groups retrieved successfully",
		Results: map[string]interface{}{
			"data": groupList,
		},
	})
}

// GetUserCommunities gets all communities for the authenticated user
func GetUserCommunities(c *fiber.Ctx) error {
	// Get user from session
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
	
	// Get user's primary device
	devices, err := userRepo.GetUserDevices(session.UserID)
	if err != nil || len(devices) == 0 {
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "No devices found",
			Results: map[string]interface{}{
				"data": []CommunityInfo{},
			},
		})
	}
	
	// Use the first device
	deviceID := devices[0].ID
	
	// Get WhatsApp client
	cm := whatsapp.GetClientManager()
	client, err := cm.GetClient(deviceID)
	if err != nil {
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Device not connected",
			Results: map[string]interface{}{
				"data": []CommunityInfo{},
			},
		})
	}
	
	// Get all groups (including communities)
	groups, err := client.GetJoinedGroups()
	if err != nil {
		logrus.Errorf("Failed to get communities: %v", err)
		return c.JSON(utils.ResponseData{
			Status:  200,
			Code:    "SUCCESS",
			Message: "Failed to get communities",
			Results: map[string]interface{}{
				"data": []CommunityInfo{},
			},
		})
	}
	
	// Filter for communities (parent groups)
	communityList := make([]CommunityInfo, 0)
	for _, group := range groups {
		if group.IsParent {
			communityInfo := CommunityInfo{
				JID:         group.JID.String(),
				Name:        group.GroupName.Name,
				Description: group.GroupTopic.Topic,
			}
			
			// Check if user is admin
			for _, participant := range group.Participants {
				if participant.JID.User == client.Store.ID.User && participant.IsAdmin {
					communityInfo.IsAdmin = true
					break
				}
			}
			
			communityList = append(communityList, communityInfo)
		}
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: "Communities retrieved successfully",
		Results: map[string]interface{}{
			"data": communityList,
		},
	})
}
