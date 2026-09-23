package masterpassword

import (
	"crypto/rand"

	"golang.org/x/crypto/argon2"

	"github.com/davidmovas/postulator/internal/adapters/secrets/aesgcm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	Version    = "v1:"
	SaltLength = 16

	timeCost    = 3
	memoryCost  = 64 * 1024
	parallelism = 4
)

func Seal(password string, plaintext []byte) ([]byte, error) {
	if password == "" {
		return nil, errors.New(errors.Invalid, "the master password must not be empty")
	}
	if len(plaintext) == 0 {
		return nil, errors.New(errors.Invalid, "nothing to seal under the master password")
	}

	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "generate the master password salt")
	}

	sealed, err := aesgcm.Seal(Derive(password, salt), plaintext)
	if err != nil {
		return nil, err
	}

	envelope := make([]byte, 0, len(Version)+SaltLength+len(sealed))
	envelope = append(envelope, Version...)
	envelope = append(envelope, salt...)
	return append(envelope, sealed...), nil
}

func Open(password string, envelope []byte) ([]byte, error) {
	if password == "" {
		return nil, errors.New(errors.Invalid, "the master password must not be empty")
	}

	header := len(Version) + SaltLength
	if len(envelope) <= header || string(envelope[:len(Version)]) != Version {
		return nil, errors.New(errors.Invalid, "the master password envelope is not readable")
	}

	opened, err := aesgcm.Open(Derive(password, envelope[len(Version):header]), envelope[header:])
	if err != nil {
		return nil, errors.New(errors.Locked, "the master password is not correct").WithInternal(err)
	}
	return opened, nil
}

func Derive(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, timeCost, memoryCost, parallelism, aesgcm.KeyLength)
}
