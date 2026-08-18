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
				if normalized == "credentials" && path == "" {
					if err := validateCredentials(item, field); err != nil {
						return err
					}
					continue
				}
				if normalized == "credentials" {
					return domainerrors.WithDetails(domainerrors.InvalidInput("secret_field_forbidden", "Secret material must be provided through top-level credentials"), map[string]any{"field": field})
				}
				if normalized == "persistrefreshtoken" {
					if _, ok := item.(bool); !ok {
						return domainerrors.WithDetails(domainerrors.InvalidInput("auth_profile_refresh_persistence_invalid", "persistRefreshToken must be boolean"), map[string]any{"field": field})
					}
					continue
				}
				if normalized == "manualtoken" || strings.Contains(normalized, "password") || strings.Contains(normalized, "clientsecret") || strings.Contains(normalized, "accesstoken") || strings.Contains(normalized, "refreshtoken") || normalized == "bearertoken" || normalized == "secret" {
					return domainerrors.WithDetails(domainerrors.InvalidInput("secret_field_forbidden", "Secret material must be provided through credentials"), map[string]any{"field": field})
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

// validateCredentials проверяет literal values и Workspace variable expressions.
func validateCredentials(value any, path string) error {
	switch references := value.(type) {
	case map[string]any:
		for name, value := range references {
			reference, ok := value.(string)
			if strings.TrimSpace(name) == "" || !ok || strings.TrimSpace(reference) == "" {
				return domainerrors.WithDetails(domainerrors.InvalidInput("credential_invalid", "Each credential must be a non-empty string"), map[string]any{"field": join(path, name)})
			}
		}
	default:
		return domainerrors.WithDetails(domainerrors.InvalidInput("credentials_invalid", "credentials must contain named string values"), map[string]any{"field": path})
	}
	return nil
}

// ValidateAuthProfile проверяет discriminated contract встроенных auth adapters.
func ValidateAuthProfile(input map[string]any) error {
	if err := requireExactKeys(input, "", "identity", "displayName", "description", "folderIdentity", "managedBy", "managedById", "meta", "active", "adapterId", "config", "credentials", "session"); err != nil {
		return err
	}
	adapterID := strings.TrimSpace(stringValue(input["adapterId"]))
	if adapterID == "" {
		return domainerrors.InvalidInput("auth_profile_adapter_required", "adapterId is required")
	}
	config, err := objectValue(input["config"], "config")
	if err != nil {
		return err
	}
	credentials, err := objectValue(input["credentials"], "credentials")
	if err != nil {
		return err
	}
	if err := validateCredentials(credentials, "credentials"); err != nil && len(credentials) > 0 {
		return err
	}

	switch adapterID {
	case "oidc":
		if err := requireExactKeys(config, "config", "issuer", "clientId", "scopes"); err != nil {
			return err
		}
		if stringValue(config["issuer"]) == "" || stringValue(config["clientId"]) == "" {
			return domainerrors.InvalidInput("oidc_config_invalid", "OIDC issuer and clientId are required")
		}
		if err := validateStringArray(config["scopes"], "config.scopes", true); err != nil {
			return err
		}
		if len(credentials) != 0 {
			return domainerrors.InvalidInput("oidc_credentials_forbidden", "OIDC profile credentials must be empty")
		}
		return validateSession(input["session"], true)
	case "oauth2-client-credentials":
		if err := requireExactKeys(config, "config", "tokenEndpoint", "clientId", "scopes", "clientAuthentication"); err != nil {
			return err
		}
		if stringValue(config["tokenEndpoint"]) == "" || stringValue(config["clientId"]) == "" {
			return domainerrors.InvalidInput("oauth2_client_credentials_config_invalid", "tokenEndpoint and clientId are required")
		}
		if err := validateStringArray(config["scopes"], "config.scopes", false); err != nil {
			return err
		}
		method := stringValue(config["clientAuthentication"])
		if method != "client_secret_basic" && method != "client_secret_post" {
			return domainerrors.InvalidInput("oauth2_client_authentication_invalid", "clientAuthentication must be client_secret_basic or client_secret_post")
		}
		if err := requireExactKeys(credentials, "credentials", "clientSecret"); err != nil {
			return err
		}
		if stringValue(credentials["clientSecret"]) == "" {
			return domainerrors.InvalidInput("oauth2_client_secret_required", "credentials.clientSecret is required")
		}
		return validateSession(input["session"], true)
	case "basic":
		if len(config) != 0 {
			return domainerrors.InvalidInput("basic_config_invalid", "Basic profile config must be empty")
		}
		if err := requireExactKeys(credentials, "credentials", "username", "password"); err != nil {
			return err
		}
		if stringValue(credentials["username"]) == "" || stringValue(credentials["password"]) == "" {
			return domainerrors.InvalidInput("basic_credentials_required", "Basic username and password are required")
		}
		return validateSession(input["session"], false)
	case "bearer":
		if len(config) != 0 {
			return domainerrors.InvalidInput("bearer_config_invalid", "Bearer profile config must be empty")
		}
		if err := requireExactKeys(credentials, "credentials", "token"); err != nil {
			return err
		}
		if stringValue(credentials["token"]) == "" {
			return domainerrors.InvalidInput("bearer_token_required", "credentials.token is required")
		}
		return validateSession(input["session"], false)
	default:
		if session, exists := input["session"]; exists && session != nil {
			return validateSession(session, true)
		}
		return nil
	}
}

func validateSession(value any, required bool) error {
	if value == nil {
		if required {
			return domainerrors.InvalidInput("auth_profile_session_required", "session is required for token-producing adapters")
		}
		return nil
	}
	if !required {
		return domainerrors.InvalidInput("auth_profile_session_forbidden", "session is not supported by this adapter")
	}
	session, err := objectValue(value, "session")
	if err != nil {
		return err
	}
	if err := requireExactKeys(session, "session", "storage", "persistRefreshToken"); err != nil {
		return err
	}
	storage := stringValue(session["storage"])
	if storage != "memory" && storage != "sessionStorage" && storage != "localStorage" {
		return domainerrors.InvalidInput("auth_profile_storage_invalid", "session.storage must be memory, sessionStorage or localStorage")
	}
	if _, ok := session["persistRefreshToken"].(bool); !ok {
		return domainerrors.InvalidInput("auth_profile_refresh_persistence_invalid", "session.persistRefreshToken must be boolean")
	}
	return nil
}

func objectValue(value any, path string) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, domainerrors.WithDetails(domainerrors.InvalidInput("object_expected", path+" must be an object"), map[string]any{"field": path})
	}
	return object, nil
}

func requireExactKeys(value map[string]any, path string, keys ...string) error {
	allowed := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		allowed[key] = struct{}{}
	}
	for key := range value {
		if _, ok := allowed[key]; !ok {
			return domainerrors.WithDetails(domainerrors.InvalidInput("auth_profile_field_unsupported", "Unsupported auth profile field"), map[string]any{"field": join(path, key)})
		}
	}
	return nil
}

func validateStringArray(value any, path string, required bool) error {
	items, ok := value.([]any)
	if !ok || (required && len(items) == 0) {
		return domainerrors.WithDetails(domainerrors.InvalidInput("string_array_invalid", path+" must be a string array"), map[string]any{"field": path})
	}
	for index, item := range items {
		if stringValue(item) == "" {
			return domainerrors.WithDetails(domainerrors.InvalidInput("string_array_invalid", path+" must contain non-empty strings"), map[string]any{"field": fmt.Sprintf("%s[%d]", path, index)})
		}
	}
	return nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

// join объединяет части пути к полю валидации.
func join(left, right string) string {
	if left == "" {
		return right
	}
	return left + "." + right
}
