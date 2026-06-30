package validator

import "github.com/go-playground/validator/v10"

type Validator interface {
	Validate(i any) error
}

type SimpleValidator struct {
	v *validator.Validate
}

func NewValidator() Validator {
	return &SimpleValidator{
		v: validator.New(),
	}
}

func (s *SimpleValidator) Validate(i any) error {
	return s.v.Struct(i)
}
