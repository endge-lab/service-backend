package access_tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/endge-lab/service-backend/internal/domain/entities"
)

const (
	// secretBytes — 256 бит случайности; base64url без padding даёт 43 символа.
	secretBytes        = 32
	encodedSecretChars = 43
	// displayPrefixChars — сколько символов случайной части показывается в списке токенов.
	displayPrefixChars = 8
)

// IsAccessTokenSecret проверяет только формат, чтобы middleware отличал PAT от JWT
// без обращения к базе. Подлинность проверяет Authenticate.
func IsAccessTokenSecret(value string) bool {
	return strings.HasPrefix(value, entities.AccessTokenSecretPrefix)
}

func validSecretFormat(value string) bool {
	if !IsAccessTokenSecret(value) {
		return false
	}
	body := strings.TrimPrefix(value, entities.AccessTokenSecretPrefix)
	if len(body) != encodedSecretChars {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(body)
	return err == nil
}

// generateSecret создаёт секрет, безопасный для показа префикс и SHA-256 хэш.
func generateSecret() (secret, displayPrefix string, hash []byte, err error) {
	random := make([]byte, secretBytes)
	if _, err = rand.Read(random); err != nil {
		return "", "", nil, err
	}
	body := base64.RawURLEncoding.EncodeToString(random)
	secret = entities.AccessTokenSecretPrefix + body
	return secret, entities.AccessTokenSecretPrefix + body[:displayPrefixChars], hashSecret(secret), nil
}

// hashSecret хэширует весь токен вместе с префиксом. Соль не нужна: секрет случайный и длинный.
func hashSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}
