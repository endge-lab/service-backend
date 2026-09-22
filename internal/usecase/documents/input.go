package documents

import (
	"bytes"
	"encoding/json"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

// CreateInput содержит проверенный транспортом JSON нового документа.
// Конкретная resource-модель формирует его до вызова application layer.
type CreateInput struct {
	payload json.RawMessage
}

// PatchInput содержит проверенный транспортом JSON частичного изменения документа.
type PatchInput struct {
	payload json.RawMessage
}

// NewCreateInputJSON создаёт application input из уже проверенного transport JSON.
func NewCreateInputJSON(payload []byte) (CreateInput, error) {
	if _, err := decodeValues(payload); err != nil {
		return CreateInput{}, err
	}
	return CreateInput{payload: append(json.RawMessage(nil), payload...)}, nil
}

// NewPatchInputJSON сохраняет точную PATCH-семантику, включая явные null.
func NewPatchInputJSON(payload []byte) (PatchInput, error) {
	if _, err := decodeValues(payload); err != nil {
		return PatchInput{}, err
	}
	return PatchInput{payload: append(json.RawMessage(nil), payload...)}, nil
}

// values декодирует входные данные в карту значений.
func (i CreateInput) values() (map[string]any, error) { return decodeValues(i.payload) }

// values декодирует входные данные в карту значений.
func (i PatchInput) values() (map[string]any, error) { return decodeValues(i.payload) }

// decodeValues декодирует входные данные документа в карту значений.
func decodeValues(payload json.RawMessage) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil || values == nil {
		return nil, domainerrors.InvalidInput("document_input_invalid", "Document input is invalid")
	}
	return values, nil
}
