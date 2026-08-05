package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/endge-lab/service-backend/internal/config"
)

var sessionCiphertextMagic = []byte{'E', 'N', 'D', 'G', 1}

type sessionEncryptionKey struct {
	id   string
	aead cipher.AEAD
}

// sessionEncryptionKeyring шифрует новые значения текущим ключом, а старые
// ключи сохраняет только для безопасной ротации уже записанных значений.
type sessionEncryptionKeyring struct {
	current sessionEncryptionKey
	keys    map[string]cipher.AEAD
	legacy  []cipher.AEAD
}

func newSessionEncryptionKeyring(cfg config.ConfiguratorAuthConfig) (*sessionEncryptionKeyring, error) {
	current, err := newSessionEncryptionKey(cfg.SessionEncryptionKeyID, cfg.SessionEncryptionKey)
	if err != nil {
		return nil, err
	}
	result := &sessionEncryptionKeyring{
		current: current,
		keys:    map[string]cipher.AEAD{current.id: current.aead},
		legacy:  []cipher.AEAD{current.aead},
	}
	for _, previousConfig := range cfg.SessionPreviousEncryptionKeys {
		previous, keyErr := newSessionEncryptionKey(previousConfig.ID, previousConfig.Key)
		if keyErr != nil {
			return nil, keyErr
		}
		if _, exists := result.keys[previous.id]; exists {
			return nil, fmt.Errorf("Configurator session encryption key id %q is duplicated", previous.id)
		}
		result.keys[previous.id] = previous.aead
		result.legacy = append(result.legacy, previous.aead)
	}
	return result, nil
}

func newSessionEncryptionKey(id, encoded string) (sessionEncryptionKey, error) {
	if !validSessionEncryptionKeyID(id) {
		return sessionEncryptionKey{}, fmt.Errorf("Configurator session encryption key id is invalid")
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return sessionEncryptionKey{}, fmt.Errorf("decode Configurator session encryption key %q: %w", id, err)
	}
	if len(key) != 32 {
		return sessionEncryptionKey{}, fmt.Errorf("Configurator session encryption key %q must contain 32 bytes", id)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return sessionEncryptionKey{}, fmt.Errorf("initialize Configurator session encryption key %q: %w", id, err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return sessionEncryptionKey{}, fmt.Errorf("initialize Configurator session AEAD key %q: %w", id, err)
	}
	return sessionEncryptionKey{id: id, aead: aead}, nil
}

func validSessionEncryptionKeyID(value string) bool {
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

func (r *sessionEncryptionKeyring) encrypt(value string) ([]byte, error) {
	if r == nil || r.current.aead == nil || len(r.current.id) == 0 || len(r.current.id) > 255 {
		return nil, fmt.Errorf("Configurator session encryption is unavailable")
	}
	header := make([]byte, 0, len(sessionCiphertextMagic)+1+len(r.current.id))
	header = append(header, sessionCiphertextMagic...)
	header = append(header, byte(len(r.current.id)))
	header = append(header, r.current.id...)
	nonce := make([]byte, r.current.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate encryption nonce: %w", err)
	}
	ciphertext := r.current.aead.Seal(nil, nonce, []byte(value), header)
	result := make([]byte, 0, len(header)+len(nonce)+len(ciphertext))
	result = append(result, header...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

func (r *sessionEncryptionKeyring) decrypt(value []byte) (string, error) {
	if r == nil {
		return "", fmt.Errorf("Configurator session encrypted value is invalid")
	}
	if len(value) >= len(sessionCiphertextMagic)+1 && string(value[:len(sessionCiphertextMagic)]) == string(sessionCiphertextMagic) {
		keyIDLength := int(value[len(sessionCiphertextMagic)])
		headerLength := len(sessionCiphertextMagic) + 1 + keyIDLength
		if keyIDLength == 0 || len(value) < headerLength {
			return "", fmt.Errorf("Configurator session encrypted value is invalid")
		}
		keyID := string(value[len(sessionCiphertextMagic)+1 : headerLength])
		aead, exists := r.keys[keyID]
		if !exists || len(value) < headerLength+aead.NonceSize() {
			return "", fmt.Errorf("Configurator session encrypted value is invalid")
		}
		nonce := value[headerLength : headerLength+aead.NonceSize()]
		plain, err := aead.Open(nil, nonce, value[headerLength+aead.NonceSize():], value[:headerLength])
		if err != nil {
			return "", fmt.Errorf("decrypt Configurator session value: %w", err)
		}
		return string(plain), nil
	}

	// До появления versioned envelope значения состояли только из nonce и
	// ciphertext. Перебор разрешённых ключей нужен для бесшовной первой ротации.
	for _, aead := range r.legacy {
		if len(value) < aead.NonceSize() {
			continue
		}
		nonce := value[:aead.NonceSize()]
		plain, err := aead.Open(nil, nonce, value[aead.NonceSize():], nil)
		if err == nil {
			return string(plain), nil
		}
	}
	return "", fmt.Errorf("Configurator session encrypted value is invalid")
}
