package ai_assistant

import (
	"github.com/endge-lab/service-backend/internal/domain/entities"
)

type CreateConversationRequest struct {
	ModelProfileID string `json:"modelProfileId" validate:"required,uuid"`
}

type ResetConversationRequest struct {
	CurrentConversationID string `json:"currentConversationId" validate:"omitempty,uuid"`
	ModelProfileID        string `json:"modelProfileId" validate:"required,uuid"`
}

type PatchConversationRequest struct {
	ModelProfileID string `json:"modelProfileId" validate:"required,uuid"`
}

type RunRequest struct {
	RequestID              string `json:"requestId" validate:"required,uuid"`
	ModelProfileID         string `json:"modelProfileId" validate:"required,uuid"`
	Prompt                 string `json:"prompt" validate:"required,max=20000"`
	InteractionID          string `json:"interactionId" validate:"omitempty,uuid"`
	ReplyToClarificationID string `json:"replyToClarificationId" validate:"omitempty,uuid"`
	SelectedCandidateID    string `json:"selectedCandidateId" validate:"omitempty,max=128"`
}

type ConversationListResponse struct {
	Items      []entities.AIConversation `json:"items"`
	NextCursor string                    `json:"nextCursor,omitempty"`
}

type MessageListResponse struct {
	Items             []entities.AIMessage      `json:"items"`
	NextCursor        string                    `json:"nextCursor,omitempty"`
	OpenClarification *entities.AIClarification `json:"openClarification,omitempty"`
}

type CapabilitiesResponse struct {
	Available bool                      `json:"available"`
	CanView   bool                      `json:"canView"`
	CanRun    bool                      `json:"canRun"`
	Reason    string                    `json:"reason,omitempty"`
	Adapters  []string                  `json:"adapters"`
	Models    []entities.AIModelProfile `json:"models"`
}

type ConversationResponse = entities.AIConversation
