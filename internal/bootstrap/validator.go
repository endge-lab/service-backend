package bootstrap

import "github.com/endge-lab/service-backend/internal/validator"

func InitValidator() validator.Validator {
	return validator.NewValidator()
}
