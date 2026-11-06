package ai

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestVaultAccountBindingAndRotation(t *testing.T) {
	old := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	next := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	t.Setenv("AI_ENCRYPTION_KEYS", `{"old":"`+old+`","next":"`+next+`"}`)
	t.Setenv("AI_ENCRYPTION_ACTIVE_KEY", "old")
	vault, err := LoadVault()
	if err != nil {
		t.Fatal(err)
	}
	sealed, version, err := vault.Seal("synthetic-private-key", "owner", "anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("synthetic-private-key")) {
		t.Fatal("plaintext persisted")
	}
	for _, binding := range [][2]string{{"other-user", "anthropic"}, {"owner", "openai"}} {
		if _, err = vault.Open(sealed, version, binding[0], binding[1]); err == nil {
			t.Fatal("swapped ciphertext accepted")
		}
	}
	t.Setenv("AI_ENCRYPTION_ACTIVE_KEY", "next")
	vault, err = LoadVault()
	if err != nil {
		t.Fatal(err)
	}
	plain, err := vault.Open(sealed, version, "owner", "anthropic")
	if err != nil || plain != "synthetic-private-key" {
		t.Fatal("rotation lost old key")
	}
	sealed[0] ^= 1
	if _, err = vault.Open(sealed, version, "owner", "anthropic"); err == nil {
		t.Fatal("tampering accepted")
	}
	_, version, err = vault.Seal("new-key", "owner", "anthropic")
	if err != nil || version != "next" {
		t.Fatal("wrong write key")
	}
}
func TestVaultRequiresExplicitValidKeys(t *testing.T) {
	t.Setenv("AI_ENCRYPTION_KEYS", "")
	t.Setenv("AI_ENCRYPTION_ACTIVE_KEY", "")
	if v, err := LoadVault(); v != nil || err != nil {
		t.Fatal("missing keys should leave BYOK unavailable")
	}
	t.Setenv("AI_ENCRYPTION_KEYS", `{"v1":"bad"}`)
	t.Setenv("AI_ENCRYPTION_ACTIVE_KEY", "v1")
	if _, err := LoadVault(); err == nil {
		t.Fatal("invalid key accepted")
	}
}
