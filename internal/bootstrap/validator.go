package bootstrap

import "github.com/endge-lab/service-template-go/internal/validator"

func InitValidator() validator.Validator {
	return validator.NewValidator()
}
