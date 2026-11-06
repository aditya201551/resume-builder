package ai

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
)

// Vault keeps versioned encryption keys outside the database. AAD binds a key
// to its account and provider, so swapping database ciphertext fails closed.
type Vault struct {
	active string
	keys   map[string]cipher.AEAD
}

func LoadVault() (*Vault, error) {
	raw, active := os.Getenv("AI_ENCRYPTION_KEYS"), os.Getenv("AI_ENCRYPTION_ACTIVE_KEY")
	if raw == "" && active == "" {
		return nil, nil
	}
	var encoded map[string]string
	if json.Unmarshal([]byte(raw), &encoded) != nil || active == "" {
		return nil, errors.New("invalid AI encryption configuration")
	}
	v := &Vault{active: active, keys: make(map[string]cipher.AEAD)}
	for version, value := range encoded {
		key, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(key) != 32 || version == "" {
			return nil, errors.New("AI encryption keys must be base64 encoded 32 byte keys")
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		v.keys[version], err = cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
	}
	if v.keys[active] == nil {
		return nil, errors.New("active AI encryption key is missing")
	}
	return v, nil
}

func (v *Vault) Seal(value, account, provider string) ([]byte, string, error) {
	if v == nil {
		return nil, "", errors.New("AI key storage unavailable")
	}
	aead := v.keys[v.active]
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}
	return aead.Seal(nonce, nonce, []byte(value), []byte(account+"\x00"+provider)), v.active, nil
}

func (v *Vault) Open(data []byte, version, account, provider string) (string, error) {
	if v == nil || v.keys[version] == nil {
		return "", errors.New("AI encryption key unavailable")
	}
	aead := v.keys[version]
	if len(data) < aead.NonceSize() {
		return "", errors.New("invalid encrypted AI key")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(account+"\x00"+provider))
	if err != nil {
		return "", errors.New("could not decrypt AI key")
	}
	return string(plain), nil
}
