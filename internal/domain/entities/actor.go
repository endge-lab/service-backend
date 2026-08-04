package entities

type Actor struct {
	ID          string `json:"id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}
