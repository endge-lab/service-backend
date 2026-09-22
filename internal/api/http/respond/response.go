// Package respond maps application results and errors to the shared HTTP envelope.
package respond

// ErrorResponse is the standard external transport envelope for HTTP handlers.
type ErrorResponse struct {
	Code    string         `json:"code" example:"validation_error"`
	Message string         `json:"message" example:"Запрос не прошёл валидацию"`
	Details map[string]any `json:"details,omitempty"`
}
