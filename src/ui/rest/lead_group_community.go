package rest

import (
	"fmt"
	"strings"
	
	"github.com/gofiber/fiber/v2"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/utils"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/repository"
	"github.com/sirupsen/logrus"
)

type UpdateLeadGroupRequest struct {
	LeadIDs   []string `json:"lead_ids"`
	GroupName string   `json:"group_name"`
	GroupJID  string   `json:"group_jid"`
}

type UpdateLeadCommunityRequest struct {
	LeadIDs       []string `json:"lead_ids"`
	CommunityName string   `json:"community_name"`
	CommunityJID  string   `json:"community_jid"`
}

// UpdateLeadGroup updates the group field for leads when they are added to a group
func UpdateLeadGroup(c *fiber.Ctx) error {
	var request UpdateLeadGroupRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
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
	
	leadRepo := repository.GetLeadRepository()
	updatedCount := 0
	
	for _, leadID := range request.LeadIDs {
		// Get the existing lead
		lead, err := leadRepo.GetLead(leadID)
		if err != nil {
			logrus.Warnf("Failed to get lead %s: %v", leadID, err)
			continue
		}
		
		// Check if lead belongs to user
		if lead.UserID != session.UserID {
			logrus.Warnf("Lead %s does not belong to user %s", leadID, session.UserID)
			continue
		}
		
		// Update group field - append if not already present
		currentGroups := []string{}
		if lead.Group != "" {
			currentGroups = strings.Split(lead.Group, ",")
		}
		
		// Check if group already exists
		groupExists := false
		for _, g := range currentGroups {
			if strings.TrimSpace(g) == request.GroupName {
				groupExists = true
				break
			}
		}
		
		if !groupExists {
			currentGroups = append(currentGroups, request.GroupName)
			lead.Group = strings.Join(currentGroups, ",")
			
			// Update the lead
			if err := leadRepo.UpdateLead(leadID, lead); err != nil {
				logrus.Errorf("Failed to update lead %s: %v", leadID, err)
				continue
			}
			updatedCount++
		}
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Updated %d leads with group %s", updatedCount, request.GroupName),
		Results: map[string]interface{}{
			"updated_count": updatedCount,
			"group_name":    request.GroupName,
		},
	})
}

// UpdateLeadCommunity updates the community field for leads when they are added to a community
func UpdateLeadCommunity(c *fiber.Ctx) error {
	var request UpdateLeadCommunityRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(utils.ResponseData{
			Status:  400,
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}
	
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
	
	leadRepo := repository.GetLeadRepository()
	updatedCount := 0
	
	for _, leadID := range request.LeadIDs {
		// Get the existing lead
		lead, err := leadRepo.GetLead(leadID)
		if err != nil {
			logrus.Warnf("Failed to get lead %s: %v", leadID, err)
			continue
		}
		
		// Check if lead belongs to user
		if lead.UserID != session.UserID {
			logrus.Warnf("Lead %s does not belong to user %s", leadID, session.UserID)
			continue
		}
		
		// Update community field - append if not already present
		currentCommunities := []string{}
		if lead.Community != "" {
			currentCommunities = strings.Split(lead.Community, ",")
		}
		
		// Check if community already exists
		communityExists := false
		for _, c := range currentCommunities {
			if strings.TrimSpace(c) == request.CommunityName {
				communityExists = true
				break
			}
		}
		
		if !communityExists {
			currentCommunities = append(currentCommunities, request.CommunityName)
			lead.Community = strings.Join(currentCommunities, ",")
			
			// Update the lead
			if err := leadRepo.UpdateLead(leadID, lead); err != nil {
				logrus.Errorf("Failed to update lead %s: %v", leadID, err)
				continue
			}
			updatedCount++
		}
	}
	
	return c.JSON(utils.ResponseData{
		Status:  200,
		Code:    "SUCCESS",
		Message: fmt.Sprintf("Updated %d leads with community %s", updatedCount, request.CommunityName),
		Results: map[string]interface{}{
			"updated_count": updatedCount,
			"community_name": request.CommunityName,
		},
	})
}
