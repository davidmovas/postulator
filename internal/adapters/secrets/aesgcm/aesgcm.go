package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	Version   = "v1:"
	KeyLength = 32
)

func Seal(key, plaintext []byte) ([]byte, error) {
	gcm, err := aead(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "generate the nonce")
	}

	envelope := make([]byte, 0, len(Version)+gcm.NonceSize()+len(plaintext)+gcm.Overhead())
	envelope = append(envelope, Version...)
	envelope = append(envelope, nonce...)
	return gcm.Seal(envelope, nonce, plaintext, nil), nil
}

func Open(key, envelope []byte) ([]byte, error) {
	gcm, err := aead(key)
	if err != nil {
		return nil, err
	}

	header := len(Version) + gcm.NonceSize()
	if len(envelope) < header+gcm.Overhead() || string(envelope[:len(Version)]) != Version {
		return nil, errors.New(errors.Invalid, "the secret envelope is not readable")
	}

	plaintext, err := gcm.Open(nil, envelope[len(Version):header], envelope[header:], nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "the secret envelope is not readable")
	}
	return plaintext, nil
}

func aead(key []byte) (cipher.AEAD, error) {
	if len(key) != KeyLength {
		return nil, errors.New(errors.Invalid, "the secret key must be 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the aead")
	}
	return gcm, nil
}
