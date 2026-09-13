package portable

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	domainerrors "github.com/endge-lab/service-backend/internal/domain/errors"
)

func TestEncryptedArtifactRoundTripAndRandomness(t *testing.T) {
	raw := json.RawMessage(`{"kind":"workspace-snapshot","credential":"secret"}`)
	first, err := encryptArtifact(raw, "correct horse battery staple")
	if err != nil {
		t.Fatalf("encrypt first artifact: %v", err)
	}
	second, err := encryptArtifact(raw, "correct horse battery staple")
	if err != nil {
		t.Fatalf("encrypt second artifact: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("two encryptions must use different salt and nonce")
	}
	if bytes.Contains(first, []byte("secret")) {
		t.Fatal("encrypted envelope leaks plaintext")
	}

	var firstEnvelope, secondEnvelope EncryptedEnvelope
	if err = json.Unmarshal(first, &firstEnvelope); err != nil {
		t.Fatalf("decode first envelope: %v", err)
	}
	if err = json.Unmarshal(second, &secondEnvelope); err != nil {
		t.Fatalf("decode second envelope: %v", err)
	}
	if firstEnvelope.KDF.Salt == secondEnvelope.KDF.Salt || firstEnvelope.Cipher.Nonce == secondEnvelope.Cipher.Nonce {
		t.Fatal("salt and nonce must both differ between exports")
	}
	plain, err := decryptArtifact(firstEnvelope, "correct horse battery staple")
	if err != nil {
		t.Fatalf("decrypt artifact: %v", err)
	}
	if !bytes.Equal(plain, raw) {
		t.Fatalf("round-trip changed payload: %s", plain)
	}
}

func TestEncryptedArtifactRejectsWrongPasswordAndTampering(t *testing.T) {
	raw := json.RawMessage(`{"kind":"workspace-snapshot"}`)
	encrypted, err := encryptArtifact(raw, "right password")
	if err != nil {
		t.Fatalf("encrypt artifact: %v", err)
	}
	var envelope EncryptedEnvelope
	if err = json.Unmarshal(encrypted, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}

	if _, err = decryptArtifact(envelope, "wrong password"); domainerrors.CodeOf(err) != "workspace_password_invalid" {
		t.Fatalf("wrong password code=%q err=%v", domainerrors.CodeOf(err), err)
	}

	tampered := envelope
	ciphertext, decodeErr := base64.RawStdEncoding.DecodeString(tampered.Ciphertext)
	if decodeErr != nil || len(ciphertext) == 0 {
		t.Fatalf("decode ciphertext: %v", decodeErr)
	}
	ciphertext[len(ciphertext)-1] ^= 0x01
	tampered.Ciphertext = base64.RawStdEncoding.EncodeToString(ciphertext)
	if _, err = decryptArtifact(tampered, "right password"); domainerrors.CodeOf(err) != "workspace_password_invalid" {
		t.Fatalf("tampered ciphertext code=%q err=%v", domainerrors.CodeOf(err), err)
	}

	tampered = envelope
	tampered.Cipher.Nonce = base64.RawStdEncoding.EncodeToString([]byte("short"))
	if _, err = decryptArtifact(tampered, "right password"); domainerrors.CodeOf(err) != "encrypted_workspace_invalid" {
		t.Fatalf("tampered nonce code=%q err=%v", domainerrors.CodeOf(err), err)
	}

	tampered = envelope
	tampered.KDF.MemoryKiB++
	if _, err = decryptArtifact(tampered, "right password"); domainerrors.CodeOf(err) != "encrypted_workspace_unsupported" {
		t.Fatalf("tampered parameters code=%q err=%v", domainerrors.CodeOf(err), err)
	}
}

func TestUnencryptedArtifactRemainsPlainJSON(t *testing.T) {
	raw := json.RawMessage(`{"kind":"workspace-snapshot"}`)
	result, err := encryptArtifact(raw, "")
	if err != nil {
		t.Fatalf("plain export: %v", err)
	}
	if !bytes.Equal(result, raw) {
		t.Fatalf("plain export changed bytes: %s", result)
	}
}
