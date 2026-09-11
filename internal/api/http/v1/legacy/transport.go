package legacy

type RebuildWorkspaceFoldersRequest struct {
	Confirmation string `json:"confirmation" validate:"required,max=160" example:"default"`
}

type RebuildWorkspaceFoldersResponse struct {
	FoldersDeleted    int `json:"foldersDeleted" example:"3"`
	FoldersCreated    int `json:"foldersCreated" example:"24"`
	DocumentsRelinked int `json:"documentsRelinked" example:"42"`
}
