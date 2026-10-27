package config

import (
	"fmt"
	"time"
)

const (
	// DefaultAccessTokenMaxTTL — максимальный срок персонального токена агента по умолчанию.
	DefaultAccessTokenMaxTTL = 30 * 24 * time.Hour
	// accessTokenMaxTTLLimit не даёт конфигурации превратить токен в бессрочный секрет.
	accessTokenMaxTTLLimit = 365 * 24 * time.Hour
)

// AccessTokenConfig задаёт ограничения персональных токенов доступа агентов.
type AccessTokenConfig struct {
	// MaxTTL — наибольший срок, который владелец может выбрать при выпуске токена.
	MaxTTL time.Duration
}

// EffectiveMaxTTL возвращает MaxTTL или значение по умолчанию для конфигураций,
// собранных вручную (например, в тестах) без этого раздела.
func (c AccessTokenConfig) EffectiveMaxTTL() time.Duration {
	if c.MaxTTL <= 0 {
		return DefaultAccessTokenMaxTTL
	}
	return c.MaxTTL
}

func loadAccessTokenConfig() (AccessTokenConfig, error) {
	value := AccessTokenConfig{MaxTTL: envDuration("ACCESS_TOKEN_MAX_TTL", DefaultAccessTokenMaxTTL)}
	if value.MaxTTL < time.Hour || value.MaxTTL > accessTokenMaxTTLLimit {
		return value, fmt.Errorf("ACCESS_TOKEN_MAX_TTL must be between 1h and 8760h")
	}
	return value, nil
}
