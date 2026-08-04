package workspaces

import (
	"encoding/json"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// InstalledIntegrationInput описывает binding глобальной интеграции в workspace.
type InstalledIntegrationInput struct {
	Identity      string         `json:"identity"`
	Version       string         `json:"version"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

// CreateInput содержит данные создания workspace.
type CreateInput struct {
	Identity              string                      `json:"identity"`
	DisplayName           string                      `json:"displayName"`
	Description           *string                     `json:"description,omitempty"`
	DataMode              string                      `json:"dataMode,omitempty"`
	Configuration         map[string]any              `json:"configuration,omitempty"`
	Meta                  map[string]any              `json:"meta,omitempty"`
	Active                *bool                       `json:"active,omitempty"`
	InstalledIntegrations []InstalledIntegrationInput `json:"installedIntegrations,omitempty"`
}

// PatchInput содержит проверенный JSON частичного изменения workspace.
type PatchInput struct{ payload json.RawMessage }

// NewPatchInputJSON сохраняет omitted и explicit null как разные состояния.
func NewPatchInputJSON(payload []byte) (PatchInput, error) {
	if _, err := jsonValues(payload); err != nil {
		return PatchInput{}, err
	}
	return PatchInput{payload: append(json.RawMessage(nil), payload...)}, nil
}

// values декодирует входные данные в карту значений.
func (i PatchInput) values() (map[string]any, error) { return jsonValues(i.payload) }

// inputValues преобразует типизированный ввод в карту значений.
func inputValues(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, domainerrors.InvalidInput("workspace_input_invalid", "Workspace input is invalid")
	}
	return jsonValues(raw)
}

// jsonValues декодирует JSON-данные в карту значений.
func jsonValues(raw []byte) (map[string]any, error) {
	values := map[string]any{}
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, domainerrors.InvalidInput("workspace_input_invalid", "Workspace input is invalid")
	}
	return values, nil
}
