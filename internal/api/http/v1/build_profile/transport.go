package build_profile

import "github.com/endge-lab/service-backend/internal/domain/entities"

type CreateRequest struct {
	Visibility string                        `json:"visibility" validate:"required,oneof=shared private"`
	Settings   entities.BuildProfileSettings `json:"settings" validate:"required"`
}

type PatchRequest struct {
	DisplayName *string                        `json:"displayName,omitempty" validate:"omitempty,max=160"`
	Visibility  *string                        `json:"visibility,omitempty" validate:"omitempty,oneof=shared private"`
	Settings    *entities.BuildProfileSettings `json:"settings,omitempty"`
}

type ListResponse struct {
	Items []entities.BuildProfile `json:"items"`
	Total int                     `json:"total"`
}

type Response = entities.BuildProfile
