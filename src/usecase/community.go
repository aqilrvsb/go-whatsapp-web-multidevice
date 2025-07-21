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
	// Ensure client is connected
	whatsapp.MustLogin(service.WaCli)
	
	// Create community with IsParent set to true
	req := whatsmeow.ReqCreateGroup{
		Name:     request.Name,
		IsParent: true, // This flag makes it a community instead of a regular group
	}
	
	// Create the community
	communityInfo, err := service.WaCli.CreateGroup(req)
	if err != nil {
		return "", fmt.Errorf("failed to create community: %v", err)
	}
	
	// Set community description if provided
	if request.Description != "" {
		err = service.WaCli.SetGroupTopic(communityInfo.JID, "", "", request.Description)
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
	whatsapp.MustLogin(service.WaCli)
	
	// Parse community JID
	communityJID, err := whatsapp.ParseJID(request.CommunityID)
	if err != nil {
		return nil, fmt.Errorf("invalid community ID: %v", err)
	}
	
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
	
	// Add participants to community (announcement group)
	participants, err := service.WaCli.UpdateGroupParticipants(communityJID, participantJIDs, whatsmeow.ParticipantChangeAdd)
	if err != nil {
		// If general error, mark all as failed
		for _, jid := range participantJIDs {
			result = append(result, domainCommunity.ParticipantStatus{
				Participant: jid.String(),
				Status:      "failed",
				Message:     err.Error(),
			})
		}
		return result, err
	}
	
	// Process results
	addedMap := make(map[string]bool)
	for _, p := range participants {
		if p.Error == 0 {
			addedMap[p.JID.String()] = true
		}
	}
	
	for _, jid := range participantJIDs {
		if addedMap[jid.String()] {
			result = append(result, domainCommunity.ParticipantStatus{
				Participant: jid.String(),
				Status:      "success",
				Message:     "Added to community successfully",
			})
		} else {
			result = append(result, domainCommunity.ParticipantStatus{
				Participant: jid.String(),
				Status:      "failed",
				Message:     "Failed to add to community",
			})
		}
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
	whatsapp.MustLogin(service.WaCli)
	
	// Parse JIDs
	communityJID, err := whatsapp.ParseJID(request.CommunityID)
	if err != nil {
		return fmt.Errorf("invalid community ID: %v", err)
	}
	
	groupJID, err := whatsapp.ParseJID(request.GroupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %v", err)
	}
	
	// Link group to community
	err = service.WaCli.LinkGroupToParent(groupJID, communityJID)
	if err != nil {
		return fmt.Errorf("failed to link group to community: %v", err)
	}
	
	return nil
}

// UnlinkGroupFromCommunity unlinks a group from a community
func (service *communityService) UnlinkGroupFromCommunity(ctx context.Context, request domainCommunity.UnlinkGroupRequest) error {
	whatsapp.MustLogin(service.WaCli)
	
	// Parse group JID
	groupJID, err := whatsapp.ParseJID(request.GroupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %v", err)
	}
	
	// Unlink group from community (set parent to empty JID)
	err = service.WaCli.UnlinkGroupFromParent(groupJID)
	if err != nil {
		return fmt.Errorf("failed to unlink group from community: %v", err)
	}
	
	return nil
}
