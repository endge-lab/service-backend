package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

type EncryptionKeyConfig struct {
	ID  string
	Key string
}

// EncryptionConfig is shared by browser sessions and encrypted provider credentials.
type EncryptionConfig struct {
	KeyID        string
	Key          string
	PreviousKeys []EncryptionKeyConfig
}

func loadEncryptionConfig() (EncryptionConfig, error) {
	previousKeys, err := parseEncryptionKeys(os.Getenv("ENCRYPTION_PREVIOUS_KEYS"))
	if err != nil {
		return EncryptionConfig{}, err
	}
	config := EncryptionConfig{
		KeyID:        env("ENCRYPTION_KEY_ID", "v1"),
		Key:          strings.TrimSpace(os.Getenv("ENCRYPTION_KEY")),
		PreviousKeys: previousKeys,
	}
	if err := config.Validate(); err != nil {
		return EncryptionConfig{}, err
	}
	return config, nil
}

func parseEncryptionKeys(value string) ([]EncryptionKeyConfig, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	result := make([]EncryptionKeyConfig, 0)
	for _, raw := range strings.Split(value, ",") {
		parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("ENCRYPTION_PREVIOUS_KEYS must use key-id:base64-key entries")
		}
		result = append(result, EncryptionKeyConfig{ID: strings.TrimSpace(parts[0]), Key: strings.TrimSpace(parts[1])})
	}
	return result, nil
}

func (c EncryptionConfig) Validate() error {
	decoded, err := base64.StdEncoding.DecodeString(c.Key)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}
	if !validEncryptionKeyID(c.KeyID) {
		return fmt.Errorf("ENCRYPTION_KEY_ID must contain only letters, digits, dot, underscore or hyphen")
	}
	seen := map[string]struct{}{c.KeyID: {}}
	for _, previous := range c.PreviousKeys {
		if !validEncryptionKeyID(previous.ID) {
			return fmt.Errorf("previous encryption key id %q is invalid", previous.ID)
		}
		if _, exists := seen[previous.ID]; exists {
			return fmt.Errorf("encryption key id %q is duplicated", previous.ID)
		}
		seen[previous.ID] = struct{}{}
		decoded, decodeErr := base64.StdEncoding.DecodeString(previous.Key)
		if decodeErr != nil || len(decoded) != 32 {
			return fmt.Errorf("previous encryption key %q must be a base64-encoded 32-byte key", previous.ID)
		}
	}
	return nil
}

func validEncryptionKeyID(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '.' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}
