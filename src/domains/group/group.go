package group

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow"
)

type IGroupUsecase interface {
	JoinGroupWithLink(ctx context.Context, request JoinGroupWithLinkRequest) (groupID string, err error)
	LeaveGroup(ctx context.Context, request LeaveGroupRequest) (err error)
	CreateGroup(ctx context.Context, request CreateGroupRequest) (groupID string, err error)
	ManageParticipant(ctx context.Context, request ParticipantRequest) (result []ParticipantStatus, err error)
	GetGroupRequestParticipants(ctx context.Context, request GetGroupRequestParticipantsRequest) (result []GetGroupRequestParticipantsResponse, err error)
	ManageGroupRequestParticipants(ctx context.Context, request GroupRequestParticipantsRequest) (result []ParticipantStatus, err error)
}

type JoinGroupWithLinkRequest struct {
	DeviceID string `json:"device_id" form:"device_id"`
	Link     string `json:"link" form:"link"`
}

type LeaveGroupRequest struct {
	DeviceID string `json:"device_id" form:"device_id"`
	GroupID  string `json:"group_id" form:"group_id"`
}

type CreateGroupRequest struct {
	DeviceID     string   `json:"device_id" form:"device_id"`
	Title        string   `json:"title" form:"title"`
	Participants []string `json:"participants" form:"participants"`
}

type ParticipantRequest struct {
	DeviceID     string                      `json:"device_id" form:"device_id"`
	GroupID      string                      `json:"group_id" form:"group_id"`
	Participants []string                    `json:"participants" form:"participants"`
	Action       whatsmeow.ParticipantChange `json:"action" form:"action"`
}

type ParticipantStatus struct {
	Participant string `json:"participant"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

type GetGroupRequestParticipantsRequest struct {
	DeviceID string `json:"device_id" query:"device_id"`
	GroupID  string `json:"group_id" query:"group_id"`
}

type GetGroupRequestParticipantsResponse struct {
	JID         string    `json:"jid"`
	RequestedAt time.Time `json:"requested_at"`
}

type GroupRequestParticipantsRequest struct {
	DeviceID     string                             `json:"device_id" form:"device_id"`
	GroupID      string                             `json:"group_id" form:"group_id"`
	Participants []string                           `json:"participants" form:"participants"`
	Action       whatsmeow.ParticipantRequestChange `json:"action" form:"action"`
}
