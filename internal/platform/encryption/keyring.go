package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

var ciphertextMagic = []byte{'E', 'N', 'D', 'G', 1}

type KeyConfig struct {
	ID  string
	Key string
}

type Config struct {
	Current  KeyConfig
	Previous []KeyConfig
}

type key struct {
	id   string
	aead cipher.AEAD
}

// Keyring owns the shared AES-GCM envelope. External AAD separates purposes
// without changing the legacy session ciphertext format when AAD is empty.
type Keyring struct {
	current key
	keys    map[string]cipher.AEAD
	legacy  []cipher.AEAD
}

func NewKeyring(config Config) (*Keyring, error) {
	current, err := newKey(config.Current)
	if err != nil {
		return nil, err
	}
	result := &Keyring{current: current, keys: map[string]cipher.AEAD{current.id: current.aead}, legacy: []cipher.AEAD{current.aead}}
	for _, previousConfig := range config.Previous {
		previous, err := newKey(previousConfig)
		if err != nil {
			return nil, err
		}
		if _, exists := result.keys[previous.id]; exists {
			return nil, fmt.Errorf("encryption key id %q is duplicated", previous.id)
		}
		result.keys[previous.id] = previous.aead
		result.legacy = append(result.legacy, previous.aead)
	}
	return result, nil
}

func (r *Keyring) Encrypt(value string, additionalData []byte) ([]byte, error) {
	if r == nil || r.current.aead == nil || len(r.current.id) == 0 || len(r.current.id) > 255 {
		return nil, fmt.Errorf("encryption keyring is unavailable")
	}
	header := make([]byte, 0, len(ciphertextMagic)+1+len(r.current.id))
	header = append(header, ciphertextMagic...)
	header = append(header, byte(len(r.current.id)))
	header = append(header, r.current.id...)
	nonce := make([]byte, r.current.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate encryption nonce: %w", err)
	}
	aad := appendAAD(header, additionalData)
	ciphertext := r.current.aead.Seal(nil, nonce, []byte(value), aad)
	result := make([]byte, 0, len(header)+len(nonce)+len(ciphertext))
	result = append(result, header...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

func (r *Keyring) Decrypt(value, additionalData []byte) (string, error) {
	if r == nil {
		return "", fmt.Errorf("encrypted value is invalid")
	}
	if len(value) >= len(ciphertextMagic)+1 && string(value[:len(ciphertextMagic)]) == string(ciphertextMagic) {
		keyIDLength := int(value[len(ciphertextMagic)])
		headerLength := len(ciphertextMagic) + 1 + keyIDLength
		if keyIDLength == 0 || len(value) < headerLength {
			return "", fmt.Errorf("encrypted value is invalid")
		}
		keyID := string(value[len(ciphertextMagic)+1 : headerLength])
		aead, exists := r.keys[keyID]
		if !exists || len(value) < headerLength+aead.NonceSize() {
			return "", fmt.Errorf("encrypted value is invalid")
		}
		nonce := value[headerLength : headerLength+aead.NonceSize()]
		plain, err := aead.Open(nil, nonce, value[headerLength+aead.NonceSize():], appendAAD(value[:headerLength], additionalData))
		if err != nil {
			return "", fmt.Errorf("decrypt encrypted value: %w", err)
		}
		return string(plain), nil
	}
	if len(additionalData) != 0 {
		return "", fmt.Errorf("legacy encrypted value cannot be used for this purpose")
	}
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
	return "", fmt.Errorf("encrypted value is invalid")
}

func newKey(config KeyConfig) (key, error) {
	if !ValidKeyID(config.ID) {
		return key{}, fmt.Errorf("encryption key id is invalid")
	}
	decoded, err := base64.StdEncoding.DecodeString(config.Key)
	if err != nil || len(decoded) != 32 {
		return key{}, fmt.Errorf("encryption key %q must be a base64-encoded 32-byte key", config.ID)
	}
	block, err := aes.NewCipher(decoded)
	if err != nil {
		return key{}, fmt.Errorf("initialize encryption key %q: %w", config.ID, err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return key{}, fmt.Errorf("initialize encryption AEAD key %q: %w", config.ID, err)
	}
	return key{id: config.ID, aead: aead}, nil
}

func ValidKeyID(value string) bool {
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

func appendAAD(header, additional []byte) []byte {
	if len(additional) == 0 {
		return header
	}
	result := make([]byte, 0, len(header)+1+len(additional))
	result = append(result, header...)
	result = append(result, 0)
	result = append(result, additional...)
	return result
}
