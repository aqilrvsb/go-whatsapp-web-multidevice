package community

import (
	"context"
	
	"go.mau.fi/whatsmeow/types"
)

type ICommunityUsecase interface {
	CreateCommunity(ctx context.Context, request CreateCommunityRequest) (communityID string, err error)
	AddParticipantsToCommunity(ctx context.Context, request AddParticipantsRequest) (result []ParticipantStatus, err error)
	GetCommunityInfo(ctx context.Context, request GetCommunityInfoRequest) (info *types.GroupInfo, err error)
	LinkGroupToCommunity(ctx context.Context, request LinkGroupRequest) (err error)
	UnlinkGroupFromCommunity(ctx context.Context, request UnlinkGroupRequest) (err error)
}

type CreateCommunityRequest struct {
	DeviceID    string   `json:"device_id" form:"device_id"`
	Name        string   `json:"name" form:"name"`
	Description string   `json:"description" form:"description"`
}

type AddParticipantsRequest struct {
	DeviceID     string   `json:"device_id" form:"device_id"`
	CommunityID  string   `json:"community_id" form:"community_id"`
	Participants []string `json:"participants" form:"participants"`
}

type ParticipantStatus struct {
	Participant string `json:"participant"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

type GetCommunityInfoRequest struct {
	CommunityID string `json:"community_id" query:"community_id"`
}

type LinkGroupRequest struct {
	CommunityID string `json:"community_id" form:"community_id"`
	GroupID     string `json:"group_id" form:"group_id"`
}

type UnlinkGroupRequest struct {
	CommunityID string `json:"community_id" form:"community_id"`
	GroupID     string `json:"group_id" form:"group_id"`
}
