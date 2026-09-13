package portable

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
	"golang.org/x/crypto/argon2"
)

const (
	encryptedArtifactKind = "endge-encrypted-workspace"
	envelopeVersion       = 1
	argonTime             = uint32(3)
	argonMemoryKiB        = uint32(64 * 1024)
	argonThreads          = uint8(2)
	derivedKeyLength      = uint32(32)
)

type EncryptedEnvelope struct {
	Kind       string         `json:"kind"`
	Version    int            `json:"version"`
	KDF        EnvelopeKDF    `json:"kdf"`
	Cipher     EnvelopeCipher `json:"cipher"`
	Ciphertext string         `json:"ciphertext"`
}

type EnvelopeKDF struct {
	Algorithm string `json:"algorithm"`
	Salt      string `json:"salt"`
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memoryKiB"`
	Threads   uint8  `json:"threads"`
	KeyLength uint32 `json:"keyLength"`
}

type EnvelopeCipher struct {
	Algorithm string `json:"algorithm"`
	Nonce     string `json:"nonce"`
}

func encryptArtifact(raw []byte, password string) (json.RawMessage, error) {
	if password == "" {
		return raw, nil
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate export salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, derivedKeyLength)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate export nonce: %w", err)
	}
	envelope := EncryptedEnvelope{
		Kind: encryptedArtifactKind, Version: envelopeVersion,
		KDF: EnvelopeKDF{
			Algorithm: "argon2id", Salt: base64.RawStdEncoding.EncodeToString(salt), Time: argonTime,
			MemoryKiB: argonMemoryKiB, Threads: argonThreads, KeyLength: derivedKeyLength,
		},
		Cipher: EnvelopeCipher{Algorithm: "aes-256-gcm", Nonce: base64.RawStdEncoding.EncodeToString(nonce)},
	}
	aad, _ := json.Marshal(struct {
		Kind    string         `json:"kind"`
		Version int            `json:"version"`
		KDF     EnvelopeKDF    `json:"kdf"`
		Cipher  EnvelopeCipher `json:"cipher"`
	}{envelope.Kind, envelope.Version, envelope.KDF, envelope.Cipher})
	envelope.Ciphertext = base64.RawStdEncoding.EncodeToString(aead.Seal(nil, nonce, raw, aad))
	return json.Marshal(envelope)
}

func decryptArtifact(envelope EncryptedEnvelope, password string) (json.RawMessage, error) {
	if envelope.Kind != encryptedArtifactKind || envelope.Version != envelopeVersion ||
		envelope.KDF.Algorithm != "argon2id" || envelope.Cipher.Algorithm != "aes-256-gcm" ||
		envelope.KDF.Time != argonTime || envelope.KDF.MemoryKiB != argonMemoryKiB ||
		envelope.KDF.Threads != argonThreads || envelope.KDF.KeyLength != derivedKeyLength {
		return nil, domainerrors.InvalidInput("encrypted_workspace_unsupported", "Encrypted workspace envelope is not supported")
	}
	if password == "" {
		return nil, domainerrors.InvalidInput("workspace_password_required", "Password is required for encrypted workspace import")
	}
	salt, saltErr := base64.RawStdEncoding.DecodeString(envelope.KDF.Salt)
	nonce, nonceErr := base64.RawStdEncoding.DecodeString(envelope.Cipher.Nonce)
	ciphertext, ciphertextErr := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if saltErr != nil || nonceErr != nil || ciphertextErr != nil || len(salt) != 16 {
		return nil, domainerrors.InvalidInput("encrypted_workspace_invalid", "Encrypted workspace envelope is invalid")
	}
	key := argon2.IDKey([]byte(password), salt, envelope.KDF.Time, envelope.KDF.MemoryKiB, envelope.KDF.Threads, envelope.KDF.KeyLength)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, domainerrors.InvalidInput("workspace_password_invalid", "Password or encrypted workspace artifact is invalid")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != aead.NonceSize() {
		return nil, domainerrors.InvalidInput("encrypted_workspace_invalid", "Encrypted workspace envelope is invalid")
	}
	aad, _ := json.Marshal(struct {
		Kind    string         `json:"kind"`
		Version int            `json:"version"`
		KDF     EnvelopeKDF    `json:"kdf"`
		Cipher  EnvelopeCipher `json:"cipher"`
	}{envelope.Kind, envelope.Version, envelope.KDF, envelope.Cipher})
	plain, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, domainerrors.InvalidInput("workspace_password_invalid", "Password or encrypted workspace artifact is invalid")
	}
	return plain, nil
}

func artifactKind(raw []byte) string {
	var header struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(raw, &header)
	return strings.TrimSpace(header.Kind)
}
