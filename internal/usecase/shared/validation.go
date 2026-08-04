package shared

import (
	"fmt"
	"strings"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

var readOnlyFields = []string{"id", "type", "revision", "author", "createdBy", "updatedBy", "createdAt", "updatedAt", "deletedAt", "created_by", "updated_by"}

// RejectReadOnly не позволяет transport payload подменить actor и audit поля.
func RejectReadOnly(input map[string]any) error {
	for _, field := range readOnlyFields {
		if _, exists := input[field]; exists {
			return domainerrors.WithDetails(domainerrors.InvalidInput("read_only_field", "Actor and audit fields are read-only"), map[string]any{"field": field})
		}
	}
	return nil
}

// ValidateSecrets рекурсивно запрещает сохранять credential material в публичном JSON.
func ValidateSecrets(value any) error {
	var walk func(any, string) error
	walk = func(current any, path string) error {
		switch typed := current.(type) {
		case map[string]any:
			for key, item := range typed {
				normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
				field := join(path, key)
				if normalized == "credentialrefs" {
					if err := validateCredentialRefs(item, field); err != nil {
						return err
					}
					continue
				}
				if normalized == "manualtoken" || strings.Contains(normalized, "password") || strings.Contains(normalized, "clientsecret") || strings.Contains(normalized, "accesstoken") || strings.Contains(normalized, "refreshtoken") || normalized == "bearertoken" || normalized == "secret" {
					return domainerrors.WithDetails(domainerrors.InvalidInput("secret_field_forbidden", "Secret material must be provided through credentialRefs"), map[string]any{"field": field})
				}
				if err := walk(item, field); err != nil {
					return err
				}
			}
		case []any:
			for index, item := range typed {
				if err := walk(item, fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(value, "")
}

// validateCredentialRefs проверяет корректность ссылок на учётные данные.
func validateCredentialRefs(value any, path string) error {
	switch references := value.(type) {
	case map[string]any:
		for name, value := range references {
			reference, ok := value.(string)
			if strings.TrimSpace(name) == "" || !ok || strings.TrimSpace(reference) == "" {
				return domainerrors.WithDetails(domainerrors.InvalidInput("credential_ref_invalid", "Each credential reference must be a non-empty string"), map[string]any{"field": join(path, name)})
			}
		}
	case []any:
		for index, value := range references {
			reference, ok := value.(string)
			if !ok || strings.TrimSpace(reference) == "" {
				return domainerrors.WithDetails(domainerrors.InvalidInput("credential_ref_invalid", "Each credential reference must be a non-empty string"), map[string]any{"field": fmt.Sprintf("%s[%d]", path, index)})
			}
		}
	default:
		return domainerrors.WithDetails(domainerrors.InvalidInput("credential_refs_invalid", "credentialRefs must contain named external references"), map[string]any{"field": path})
	}
	return nil
}

// join объединяет части пути к полю валидации.
func join(left, right string) string {
	if left == "" {
		return right
	}
	return left + "." + right
}
