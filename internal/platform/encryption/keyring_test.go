package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestKeyRotationReadsOldCiphertext(t *testing.T) {
	oldKeyring, err := NewKeyring(Config{Current: KeyConfig{ID: "v1", Key: encodedTestKey(1)}})
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := oldKeyring.Encrypt("refresh-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := NewKeyring(Config{
		Current:  KeyConfig{ID: "v2", Key: encodedTestKey(2)},
		Previous: []KeyConfig{{ID: "v1", Key: encodedTestKey(1)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rotated.Decrypt(ciphertext, nil)
	if err != nil || plain != "refresh-token" {
		t.Fatalf("old ciphertext was not decrypted: value=%q err=%v", plain, err)
	}
}

func TestKeyringReadsLegacySessionCiphertext(t *testing.T) {
	oldKeyring, err := NewKeyring(Config{Current: KeyConfig{ID: "v1", Key: encodedTestKey(3)}})
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, oldKeyring.current.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	legacy := oldKeyring.current.aead.Seal(nonce, nonce, []byte("legacy-refresh-token"), nil)
	rotated, err := NewKeyring(Config{
		Current:  KeyConfig{ID: "v2", Key: encodedTestKey(4)},
		Previous: []KeyConfig{{ID: "v1", Key: encodedTestKey(3)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := rotated.Decrypt(legacy, nil)
	if err != nil || plain != "legacy-refresh-token" {
		t.Fatalf("legacy ciphertext was not decrypted: value=%q err=%v", plain, err)
	}
}

func TestKeyringSeparatesPurposesAndRejectsTampering(t *testing.T) {
	keyring, err := NewKeyring(Config{Current: KeyConfig{ID: "v1", Key: encodedTestKey(5)}})
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := keyring.Encrypt("credential", []byte("ai-provider:connection-a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := keyring.Decrypt(ciphertext, []byte("auth-session")); err == nil {
		t.Fatal("ciphertext was accepted for another purpose")
	}
	tampered := append([]byte(nil), ciphertext...)
	tampered[len(tampered)-1] ^= 0xff
	if _, err := keyring.Decrypt(tampered, []byte("ai-provider:connection-a")); err == nil {
		t.Fatal("tampered ciphertext was accepted")
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
