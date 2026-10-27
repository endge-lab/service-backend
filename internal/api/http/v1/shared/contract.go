package shared

import (
	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	appvalidator "github.com/endge-lab/service-kit-go/pkg/validator"
)

// RequestContract проверяет JSON-тело правилами одного REST DTO.
// Нужен другим transport adapters (MCP), которые принимают те же данные, что REST,
// но получают их не из fiber.Ctx.
type RequestContract struct {
	validate func([]byte) error
}

// NewRequestContract связывает DTO-тип Request с общим валидатором.
func NewRequestContract[Request any](validator appvalidator.Validator) RequestContract {
	return RequestContract{validate: func(body []byte) error {
		_, err := DecodeAndValidateJSON[Request](body, validator)
		return err
	}}
}

// Validate проверяет тело так же, как REST-обработчик ресурса.
// Пустой контракт (не собранный NewRequestContract) отклоняет любое тело.
func (c RequestContract) Validate(body []byte) error {
	if c.validate == nil {
		return domainerrors.Internal("request_contract_missing", "Request contract is not configured")
	}
	return c.validate(body)
}

// DocumentContract — DTO создания и частичного изменения документа одного ресурса,
// те же типы, что передаются в CreateDocument и PatchDocument.
type DocumentContract struct {
	Create RequestContract
	Patch  RequestContract
}

// NewDocumentContract собирает контракт ресурса из его CreateRequest и PatchRequest.
func NewDocumentContract[Create, Patch any](validator appvalidator.Validator) DocumentContract {
	return DocumentContract{Create: NewRequestContract[Create](validator), Patch: NewRequestContract[Patch](validator)}
}
