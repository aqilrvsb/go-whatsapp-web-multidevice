package usecase

import (
	"context"
	"fmt"
	
	domainCommunity "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/community"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type communityService struct {
	WaCli *whatsmeow.Client
}

func NewCommunityService(waCli *whatsmeow.Client) domainCommunity.ICommunityUsecase {
	return &communityService{
		WaCli: waCli,
	}
}

// CreateCommunity creates a new WhatsApp community
func (service *communityService) CreateCommunity(ctx context.Context, request domainCommunity.CreateCommunityRequest) (communityID string, err error) {
	// Get device-specific client
	var waClient *whatsmeow.Client
	if request.DeviceID != "" {
		cm := whatsapp.GetClientManager()
		waClient, err = cm.GetClient(request.DeviceID)
		if err != nil {
			return "", fmt.Errorf("device not connected: %v", err)
		}
	} else {
		// Fallback to default client
		waClient = service.WaCli
	}
	
	// Ensure client is connected
	whatsapp.MustLogin(waClient)
	
	// Create community with GroupParent configuration
	req := whatsmeow.ReqCreateGroup{
		Name:        request.Name,
		GroupParent: types.GroupParent{IsParent: true},
	}
	
	// Create the community
	communityInfo, err := waClient.CreateGroup(req)
	if err != nil {
		return "", fmt.Errorf("failed to create community: %v", err)
	}
	
	// Set community description if provided
	if request.Description != "" {
		err = waClient.SetGroupTopic(communityInfo.JID, "", "", request.Description)
		if err != nil {
			// Log error but don't fail the community creation
			fmt.Printf("Warning: Failed to set community description: %v\n", err)
		}
	}
	
	return communityInfo.JID.String(), nil
}

// AddParticipantsToCommunity adds participants to a community
// This actually adds them to the community's announcement group
func (service *communityService) AddParticipantsToCommunity(ctx context.Context, request domainCommunity.AddParticipantsRequest) (result []domainCommunity.ParticipantStatus, err error) {
	// Get device-specific client
	var waClient *whatsmeow.Client
	if request.DeviceID != "" {
		cm := whatsapp.GetClientManager()
		waClient, err = cm.GetClient(request.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("device not connected: %v", err)
		}
	} else {
		// Fallback to default client
		waClient = service.WaCli
	}
	
	whatsapp.MustLogin(waClient)
	
	// Parse community JID
	communityJID, err := whatsapp.ParseJID(request.CommunityID)
	if err != nil {
		return nil, fmt.Errorf("invalid community ID: %v", err)
	}
	
	// Get community info to find the announcement group
	communityInfo, err := waClient.GetGroupInfo(communityJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get community info: %v", err)
	}
	
	// Check if this is actually a community
	if !communityInfo.IsParent {
		return nil, fmt.Errorf("the provided ID is not a community")
	}
	
	// Find the announcement group (default group) of the community
	// In WhatsApp communities, the announcement group is typically the default linked group
	var announcementGroupJID types.JID
	
	// Get linked groups
	subgroups, err := waClient.GetSubGroups(communityJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get community subgroups: %v", err)
	}
	
	// The announcement group is usually the first linked group or has a special indicator
	if len(subgroups) == 0 {
		return nil, fmt.Errorf("no announcement group found for this community")
	}
	
	// Use the first subgroup as the announcement group
	// In most cases, this is the default group created with the community
	announcementGroupJID = subgroups[0].JID
	
	// Parse participant JIDs
	var participantJIDs []types.JID
	result = make([]domainCommunity.ParticipantStatus, 0, len(request.Participants))
	
	for _, participant := range request.Participants {
		jid, err := whatsapp.ParseJID(participant)
		if err != nil {
			result = append(result, domainCommunity.ParticipantStatus{
				Participant: participant,
				Status:      "failed",
				Message:     fmt.Sprintf("Invalid phone number format: %v", err),
			})
			continue
		}
		participantJIDs = append(participantJIDs, jid)
	}
	
	if len(participantJIDs) == 0 {
		return result, fmt.Errorf("no valid participants to add")
	}
	
	// Add participants to the announcement group
	participants, err := waClient.UpdateGroupParticipants(announcementGroupJID, participantJIDs, whatsmeow.ParticipantChangeAdd)
	if err != nil {
		// If general error, mark all as failed
		for _, jid := range participantJIDs {
			result = append(result, domainCommunity.ParticipantStatus{
				Participant: jid.String(),
				Status:      "failed", 
				Message:     fmt.Sprintf("Failed to add to community: %v", err),
			})
		}
		return result, fmt.Errorf("failed to add participants: %v", err)
	}
	
	// Process results
	for _, p := range participants {
		status := "failed"
		message := "Failed to add participant"
		
		if p.Error == 0 {
			status = "success"
			message = "Added to community announcement group successfully"
		} else if p.Error == 403 {
			message = "Permission denied - user may have privacy settings blocking adds"
		} else if p.Error == 409 {
			message = "User is already in the group"
		} else {
			message = fmt.Sprintf("Failed with error code: %d", p.Error)
		}
		
		result = append(result, domainCommunity.ParticipantStatus{
			Participant: p.JID.String(),
			Status:      status,
			Message:     message,
		})
	}
	
	return result, nil
}

// GetCommunityInfo retrieves information about a community
func (service *communityService) GetCommunityInfo(ctx context.Context, request domainCommunity.GetCommunityInfoRequest) (info *types.GroupInfo, err error) {
	whatsapp.MustLogin(service.WaCli)
	
	// Parse community JID
	communityJID, err := whatsapp.ParseJID(request.CommunityID)
	if err != nil {
		return nil, fmt.Errorf("invalid community ID: %v", err)
	}
	
	// Get group info (communities are special groups)
	groupInfo, err := service.WaCli.GetGroupInfo(communityJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get community info: %v", err)
	}
	
	return groupInfo, nil
}

// LinkGroupToCommunity links an existing group to a community
func (service *communityService) LinkGroupToCommunity(ctx context.Context, request domainCommunity.LinkGroupRequest) error {
	// TODO: This functionality may not be available in the current whatsmeow version
	// The API method LinkGroupToParent might need to be implemented or may be available in newer versions
	return fmt.Errorf("linking groups to communities is not yet supported in this version")
}

// UnlinkGroupFromCommunity unlinks a group from a community
func (service *communityService) UnlinkGroupFromCommunity(ctx context.Context, request domainCommunity.UnlinkGroupRequest) error {
	// TODO: This functionality may not be available in the current whatsmeow version
	// The API method UnlinkGroupFromParent might need to be implemented or may be available in newer versions
	return fmt.Errorf("unlinking groups from communities is not yet supported in this version")
}
