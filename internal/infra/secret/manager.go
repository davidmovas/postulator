package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

type Manager struct {
	keyPath string
	key     []byte
}

func NewManager() (*Manager, error) {
	m := &Manager{keyPath: filepath.Join(xdg.ConfigHome, "Postulator", "secret.key")}
	if err := m.ensureKey(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) ensureKey() error {
	if m.key != nil {
		return nil
	}
	if _, err := os.Stat(m.keyPath); err == nil {
		var b []byte
		b, err = os.ReadFile(m.keyPath)
		if err != nil {
			return err
		}
		m.key = b
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(m.keyPath), 0o755); err != nil {
		return err
	}

	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return err
	}

	if err := os.WriteFile(m.keyPath, key, 0o600); err != nil {
		return err
	}
	m.key = key
	return nil
}

// Encrypt returns base64(nonce|ciphertext) for the given plaintext.
func (m *Manager) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	buf := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(buf), nil
}

// Decrypt accepts base64(nonce|ciphertext) and returns plaintext.
func (m *Manager) Decrypt(enc string) (string, error) {
	if enc == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("invalid ciphertext")
	}
	nonce := raw[:ns]
	ct := raw[ns:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
