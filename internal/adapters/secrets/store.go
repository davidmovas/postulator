package secrets

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/secrets/aesgcm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type vault interface {
	Put(ctx context.Context, ref string, ciphertext []byte) error
	Get(ctx context.Context, ref string) ([]byte, error)
	Delete(ctx context.Context, ref string) error
}

type Store struct {
	vault vault
	key   []byte
}

func NewStore(v vault, key []byte) *Store {
	return &Store{vault: v, key: key}
}

func (s *Store) Put(ctx context.Context, ref, value string) error {
	if ref == "" {
		return errors.New(errors.Invalid, "secret reference must not be empty")
	}

	sealed, err := aesgcm.Seal(s.key, []byte(value))
	if err != nil {
		return err
	}
	return s.vault.Put(ctx, ref, sealed)
}

func (s *Store) Get(ctx context.Context, ref string) (string, error) {
	sealed, err := s.vault.Get(ctx, ref)
	if err != nil {
		return "", err
	}

	plaintext, err := aesgcm.Open(s.key, sealed)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *Store) Delete(ctx context.Context, ref string) error {
	return s.vault.Delete(ctx, ref)
}
