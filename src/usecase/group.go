package usecase

import (
	"context"
	"fmt"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domainGroup "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/group"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/validations"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

type serviceGroup struct {
	WaCli *whatsmeow.Client
}

func NewGroupService(waCli *whatsmeow.Client) domainGroup.IGroupUsecase {
	return &serviceGroup{
		WaCli: waCli,
	}
}

func (service serviceGroup) JoinGroupWithLink(ctx context.Context, request domainGroup.JoinGroupWithLinkRequest) (groupID string, err error) {
	if err = validations.ValidateJoinGroupWithLink(ctx, request); err != nil {
		return groupID, err
	}
	whatsapp.MustLogin(service.WaCli)

	jid, err := service.WaCli.JoinGroupWithLink(request.Link)
	if err != nil {
		return
	}
	return jid.String(), nil
}

func (service serviceGroup) LeaveGroup(ctx context.Context, request domainGroup.LeaveGroupRequest) (err error) {
	if err = validations.ValidateLeaveGroup(ctx, request); err != nil {
		return err
	}

	JID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return err
	}

	return service.WaCli.LeaveGroup(JID)
}

func (service serviceGroup) CreateGroup(ctx context.Context, request domainGroup.CreateGroupRequest) (groupID string, err error) {
	if err = validations.ValidateCreateGroup(ctx, request); err != nil {
		return groupID, err
	}
	
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
	
	whatsapp.MustLogin(waClient)

	participantsJID, err := service.participantToJID(request.Participants)
	if err != nil {
		return
	}

	groupConfig := whatsmeow.ReqCreateGroup{
		Name:              request.Title,
		Participants:      participantsJID,
		GroupParent:       types.GroupParent{},
		GroupLinkedParent: types.GroupLinkedParent{},
	}

	groupInfo, err := waClient.CreateGroup(groupConfig)
	if err != nil {
		return
	}

	return groupInfo.JID.String(), nil
}

func (service serviceGroup) ManageParticipant(ctx context.Context, request domainGroup.ParticipantRequest) (result []domainGroup.ParticipantStatus, err error) {
	if err = validations.ValidateParticipant(ctx, request); err != nil {
		return result, err
	}
	
	// Get device-specific client
	var waClient *whatsmeow.Client
	if request.DeviceID != "" {
		cm := whatsapp.GetClientManager()
		waClient, err = cm.GetClient(request.DeviceID)
		if err != nil {
			return result, fmt.Errorf("device not connected: %v", err)
		}
	} else {
		// Fallback to default client
		waClient = service.WaCli
	}
	
	whatsapp.MustLogin(waClient)

	groupJID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return result, err
	}

	participantsJID, err := service.participantToJID(request.Participants)
	if err != nil {
		return result, err
	}

	participants, err := waClient.UpdateGroupParticipants(groupJID, participantsJID, request.Action)
	if err != nil {
		return result, err
	}

	for _, participant := range participants {
		if participant.Error == 403 && participant.AddRequest != nil {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "error",
				Message:     "Failed to add participant",
			})
		} else {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "success",
				Message:     "Action success",
			})
		}
	}

	return result, nil
}

func (service serviceGroup) GetGroupRequestParticipants(ctx context.Context, request domainGroup.GetGroupRequestParticipantsRequest) (result []domainGroup.GetGroupRequestParticipantsResponse, err error) {
	if err = validations.ValidateGetGroupRequestParticipants(ctx, request); err != nil {
		return result, err
	}

	groupJID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return result, err
	}

	participants, err := service.WaCli.GetGroupRequestParticipants(groupJID)
	if err != nil {
		return result, err
	}

	for _, participant := range participants {
		result = append(result, domainGroup.GetGroupRequestParticipantsResponse{
			JID:         participant.JID.String(),
			RequestedAt: participant.RequestedAt,
		})
	}

	return result, nil
}

func (service serviceGroup) ManageGroupRequestParticipants(ctx context.Context, request domainGroup.GroupRequestParticipantsRequest) (result []domainGroup.ParticipantStatus, err error) {
	if err = validations.ValidateManageGroupRequestParticipants(ctx, request); err != nil {
		return result, err
	}

	groupJID, err := whatsapp.ValidateJidWithLogin(service.WaCli, request.GroupID)
	if err != nil {
		return result, err
	}

	participantsJID, err := service.participantToJID(request.Participants)
	if err != nil {
		return result, err
	}

	participants, err := service.WaCli.UpdateGroupRequestParticipants(groupJID, participantsJID, request.Action)
	if err != nil {
		return result, err
	}

	for _, participant := range participants {
		if participant.Error != 0 {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "error",
				Message:     fmt.Sprintf("Action %s failed (code %d)", request.Action, participant.Error),
			})
		} else {
			result = append(result, domainGroup.ParticipantStatus{
				Participant: participant.JID.String(),
				Status:      "success",
				Message:     fmt.Sprintf("Action %s success", request.Action),
			})
		}
	}

	return result, nil
}

func (service serviceGroup) participantToJID(participants []string) ([]types.JID, error) {
	var participantsJID []types.JID
	for _, participant := range participants {
		formattedParticipant := participant + config.WhatsappTypeUser

		if !whatsapp.IsOnWhatsapp(service.WaCli, formattedParticipant) {
			return nil, pkgError.ErrUserNotRegistered
		}

		if participantJID, err := types.ParseJID(formattedParticipant); err == nil {
			participantsJID = append(participantsJID, participantJID)
		}
	}
	return participantsJID, nil
}
