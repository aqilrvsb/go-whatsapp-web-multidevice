package usecase

import (
	"context"
	"fmt"
	"time"
	
	domainGroup "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/group"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/validations"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// Enhanced group service with additional functionality
type groupServiceEnhanced struct {
	*serviceGroup
}

// CreateGroupWithParticipants creates a new group and adds participants in one operation
func (service *serviceGroup) CreateGroupWithParticipants(ctx context.Context, request domainGroup.CreateGroupRequest) (groupID string, addResults []domainGroup.ParticipantStatus, err error) {
	// Validate request
	err = validations.ValidateCreateGroup(ctx, request)
	if err != nil {
		return "", nil, err
	}
	
	whatsapp.MustLogin(service.WaCli)
	
	// First, create the group
	req := whatsmeow.ReqCreateGroup{
		Name:         request.Title,
		Participants: []types.JID{}, // Start with empty, we'll add participants after
	}
	
	groupInfo, err := service.WaCli.CreateGroup(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create group: %v", err)
	}
	
	groupID = groupInfo.JID.String()
	
	// If no participants specified, return just the group ID
	if len(request.Participants) == 0 {
		return groupID, []domainGroup.ParticipantStatus{}, nil
	}
	
	// Add participants to the newly created group
	addResults = make([]domainGroup.ParticipantStatus, 0, len(request.Participants))
	
	// Parse participant JIDs
	var participantJIDs []types.JID
	for _, participant := range request.Participants {
		jid, err := whatsapp.ParseJID(participant)
		if err != nil {
			addResults = append(addResults, domainGroup.ParticipantStatus{
				Participant: participant,
				Status:      "failed",
				Message:     fmt.Sprintf("Invalid phone number format: %v", err),
			})
			continue
		}
		participantJIDs = append(participantJIDs, jid)
	}
	
	// Add participants if we have valid JIDs
	if len(participantJIDs) > 0 {
		// Small delay to ensure group is fully created
		time.Sleep(500 * time.Millisecond)
		
		participants, err := service.WaCli.UpdateGroupParticipants(groupInfo.JID, participantJIDs, whatsmeow.ParticipantChangeAdd)
		if err != nil {
			// Log error but don't fail - group is already created
			fmt.Printf("Warning: Failed to add some participants: %v\n", err)
			
			// Mark all as failed
			for _, jid := range participantJIDs {
				addResults = append(addResults, domainGroup.ParticipantStatus{
					Participant: jid.String(),
					Status:      "failed",
					Message:     err.Error(),
				})
			}
		} else {
			// Process results
			addedMap := make(map[string]bool)
			for _, p := range participants {
				if p.Error == 0 {
					addedMap[p.JID.String()] = true
				}
			}
			
			for _, jid := range participantJIDs {
				if addedMap[jid.String()] {
					addResults = append(addResults, domainGroup.ParticipantStatus{
						Participant: jid.String(),
						Status:      "success",
						Message:     "Added to group successfully",
					})
				} else {
					addResults = append(addResults, domainGroup.ParticipantStatus{
						Participant: jid.String(),
						Status:      "failed",
						Message:     "Failed to add to group",
					})
				}
			}
		}
	}
	
	return groupID, addResults, nil
}

// GetGroupInviteLink gets the invite link for a group
func (service *serviceGroup) GetGroupInviteLink(ctx context.Context, groupID string) (inviteLink string, err error) {
	whatsapp.MustLogin(service.WaCli)
	
	// Parse group JID
	groupJID, err := whatsapp.ParseJID(groupID)
	if err != nil {
		return "", fmt.Errorf("invalid group ID: %v", err)
	}
	
	// Get invite link
	link, err := service.WaCli.GetGroupInviteLink(groupJID, false)
	if err != nil {
		return "", fmt.Errorf("failed to get group invite link: %v", err)
	}
	
	return link, nil
}

// RevokeGroupInviteLink revokes and regenerates the invite link for a group
func (service *serviceGroup) RevokeGroupInviteLink(ctx context.Context, groupID string) (newInviteLink string, err error) {
	whatsapp.MustLogin(service.WaCli)
	
	// Parse group JID
	groupJID, err := whatsapp.ParseJID(groupID)
	if err != nil {
		return "", fmt.Errorf("invalid group ID: %v", err)
	}
	
	// Revoke and get new invite link
	link, err := service.WaCli.GetGroupInviteLink(groupJID, true)
	if err != nil {
		return "", fmt.Errorf("failed to revoke group invite link: %v", err)
	}
	
	return link, nil
}

// GetAllGroups gets all groups the user is part of
func (service *serviceGroup) GetAllGroups(ctx context.Context) ([]*types.GroupInfo, error) {
	whatsapp.MustLogin(service.WaCli)
	
	// Get all groups
	groups, err := service.WaCli.GetJoinedGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %v", err)
	}
	
	return groups, nil
}

// SetGroupIcon sets the group profile picture
func (service *serviceGroup) SetGroupIcon(ctx context.Context, groupID string, imageData []byte) error {
	whatsapp.MustLogin(service.WaCli)
	
	// Parse group JID
	groupJID, err := whatsapp.ParseJID(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %v", err)
	}
	
	// Set group picture
	_, err = service.WaCli.SetGroupPhoto(groupJID, imageData)
	if err != nil {
		return fmt.Errorf("failed to set group icon: %v", err)
	}
	
	return nil
}

// SetGroupDescription sets the group description/topic
func (service *serviceGroup) SetGroupDescription(ctx context.Context, groupID string, description string) error {
	whatsapp.MustLogin(service.WaCli)
	
	// Parse group JID
	groupJID, err := whatsapp.ParseJID(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %v", err)
	}
	
	// Set group topic
	err = service.WaCli.SetGroupTopic(groupJID, "", "", description)
	if err != nil {
		return fmt.Errorf("failed to set group description: %v", err)
	}
	
	return nil
}
