package bootstrap

import (
	"github.com/endge-lab/service-backend/internal/config"
	platformencryption "github.com/endge-lab/service-backend/internal/platform/encryption"
)

func newEncryptionKeyring(cfg *config.Config) (*platformencryption.Keyring, error) {
	previous := make([]platformencryption.KeyConfig, 0, len(cfg.Encryption.PreviousKeys))
	for _, value := range cfg.Encryption.PreviousKeys {
		previous = append(previous, platformencryption.KeyConfig{ID: value.ID, Key: value.Key})
	}
	return platformencryption.NewKeyring(platformencryption.Config{
		Current:  platformencryption.KeyConfig{ID: cfg.Encryption.KeyID, Key: cfg.Encryption.Key},
		Previous: previous,
	})
}
