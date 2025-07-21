package validations

import (
	"context"
	
	domainCommunity "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/community"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func ValidateCreateCommunity(ctx context.Context, request domainCommunity.CreateCommunityRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&request.Description, validation.Length(0, 512)),
	)
	
	if err != nil {
		return pkgError.ValidationError(err.Error())
	}
	
	return nil
}

func ValidateAddParticipantsToCommunity(ctx context.Context, request domainCommunity.AddParticipantsRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.CommunityID, validation.Required),
		validation.Field(&request.Participants, validation.Required, validation.Length(1, 1000)),
	)
	
	if err != nil {
		return pkgError.ValidationError(err.Error())
	}
	
	return nil
}

func ValidateLinkGroup(ctx context.Context, request domainCommunity.LinkGroupRequest) error {
	err := validation.ValidateStructWithContext(ctx, &request,
		validation.Field(&request.CommunityID, validation.Required),
		validation.Field(&request.GroupID, validation.Required),
	)
	
	if err != nil {
		return pkgError.ValidationError(err.Error())
	}
	
	return nil
}
