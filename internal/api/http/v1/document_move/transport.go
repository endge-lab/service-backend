package document_move

import (
	"github.com/endge-lab/service-backend/internal/api/http/v1/shared"
	"github.com/endge-lab/service-backend/internal/usecase/documents"
)

type MoveDocumentRequest struct {
	Collection       string `json:"collection" validate:"required,max=160" example:"actions"`
	Identity         string `json:"identity" validate:"required,max=160" example:"open-schedule"`
	ExpectedRevision int    `json:"expectedRevision" validate:"required,min=1" example:"3"`
}

type MoveRequest struct {
	Documents      []MoveDocumentRequest `json:"documents" validate:"required,min=1,max=500,dive"`
	FolderIdentity string                `json:"folderIdentity" validate:"required,max=160" example:"schedule-actions"`
}

type MovedDocumentResponse struct {
	Collection string         `json:"collection" example:"actions"`
	Document   map[string]any `json:"document"`
}

type MoveResponse struct {
	Documents []MovedDocumentResponse `json:"documents"`
	Moved     int                     `json:"moved" example:"2"`
}

func newMoveResponse(result documents.MoveDocumentsResult) (MoveResponse, error) {
	response := MoveResponse{Documents: make([]MovedDocumentResponse, 0, len(result.Documents)), Moved: result.Moved}
	for _, item := range result.Documents {
		document, err := shared.DocumentMap(item.Document)
		if err != nil {
			return MoveResponse{}, err
		}
		response.Documents = append(response.Documents, MovedDocumentResponse{
			Collection: item.Collection,
			Document:   document,
		})
	}
	return response, nil
}
