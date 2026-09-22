package entities

import "time"

// BackendConnection описывает глобально зарегистрированный удалённый backend.
type BackendConnection struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	BaseURL   string    `json:"baseUrl"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}
