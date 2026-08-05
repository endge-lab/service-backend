package auth

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/endge-lab/service-backend/internal/config"
)

// TestSessionEncryptionKeyRotation проверяет чтение старого ciphertext после
// переключения записи на новый versioned key.
func TestSessionEncryptionKeyRotation(t *testing.T) {
	oldKey := encodedTestKey(1)
	newKey := encodedTestKey(2)
	oldKeyring, err := newSessionEncryptionKeyring(config.ConfiguratorAuthConfig{
		SessionEncryptionKeyID: "v1", SessionEncryptionKey: oldKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	oldCiphertext, err := oldKeyring.encrypt("refresh-token")
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := newSessionEncryptionKeyring(config.ConfiguratorAuthConfig{
		SessionEncryptionKeyID: "v2", SessionEncryptionKey: newKey,
		SessionPreviousEncryptionKeys: []config.SessionEncryptionKeyConfig{{ID: "v1", Key: oldKey}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rotated.decrypt(oldCiphertext)
	if err != nil || plain != "refresh-token" {
		t.Fatalf("старый ciphertext не расшифрован после ротации: value=%q err=%v", plain, err)
	}
	newCiphertext, err := rotated.encrypt("new-refresh-token")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = oldKeyring.decrypt(newCiphertext); err == nil {
		t.Fatal("старый keyring расшифровал значение, записанное новым ключом")
	}
}

// TestIdentityRefreshUsesEarliestTokenExpiry проверяет, что cached identity не
// переживёт срок exp identity token, даже если access token выдан дольше.
func TestIdentityRefreshUsesEarliestTokenExpiry(t *testing.T) {
	now := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	claimsExpiry := now.Add(5 * time.Minute)
	if got := tokenExpiry(now, 3600, claimsExpiry); !got.Equal(claimsExpiry) {
		t.Fatalf("identity refresh=%s, ожидался более ранний claims exp=%s", got, claimsExpiry)
	}
}

// TestSessionEncryptionReadsLegacyCiphertext проверяет миграционный путь для
// значений, созданных до добавления версии ключа в ciphertext envelope.
func TestSessionEncryptionReadsLegacyCiphertext(t *testing.T) {
	oldKey := encodedTestKey(3)
	oldKeyring, err := newSessionEncryptionKeyring(config.ConfiguratorAuthConfig{
		SessionEncryptionKeyID: "v1", SessionEncryptionKey: oldKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, oldKeyring.current.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	legacyCiphertext := oldKeyring.current.aead.Seal(nonce, nonce, []byte("legacy-refresh-token"), nil)

	rotated, err := newSessionEncryptionKeyring(config.ConfiguratorAuthConfig{
		SessionEncryptionKeyID: "v2", SessionEncryptionKey: encodedTestKey(4),
		SessionPreviousEncryptionKeys: []config.SessionEncryptionKeyConfig{{ID: "v1", Key: oldKey}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rotated.decrypt(legacyCiphertext)
	if err != nil || plain != "legacy-refresh-token" {
		t.Fatalf("legacy ciphertext не расшифрован: value=%q err=%v", plain, err)
	}
}

// TestSessionEncryptionRejectsTampering проверяет аутентификацию заголовка и
// содержимого encrypted envelope.
func TestSessionEncryptionRejectsTampering(t *testing.T) {
	keyring, err := newSessionEncryptionKeyring(config.ConfiguratorAuthConfig{
		SessionEncryptionKeyID: "v1", SessionEncryptionKey: encodedTestKey(5),
	})
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := keyring.encrypt("refresh-token")
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), ciphertext...)
	tampered[len(tampered)-1] ^= 0xff
	if _, err = keyring.decrypt(tampered); err == nil {
		t.Fatal("изменённый ciphertext был принят")
	}
}

func encodedTestKey(fill byte) string {
	return base64.StdEncoding.EncodeToString([]byte{
		fill, fill, fill, fill, fill, fill, fill, fill,
		fill, fill, fill, fill, fill, fill, fill, fill,
		fill, fill, fill, fill, fill, fill, fill, fill,
		fill, fill, fill, fill, fill, fill, fill, fill,
	})
}
