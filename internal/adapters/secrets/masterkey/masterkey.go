package masterkey

import (
	"crypto/rand"
	stderrors "errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/secrets/dpapi"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	FileName = "master.key"
	Length   = 32

	directoryMode = 0o700
	fileMode      = 0o600
)

func Load(dir string) ([]byte, error) {
	if dir == "" {
		return nil, errors.New(errors.Invalid, "the master key directory must not be empty")
	}

	path := filepath.Join(dir, FileName)
	protected, err := os.ReadFile(path)
	if stderrors.Is(err, fs.ErrNotExist) {
		return create(path)
	}
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "read the master key")
	}

	key, err := dpapi.Unprotect(protected)
	if err != nil {
		return nil, err
	}
	if len(key) != Length {
		return nil, errors.New(errors.Internal, "the master key has the wrong length")
	}
	return key, nil
}

func create(path string) ([]byte, error) {
	if err := os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "create the master key directory")
	}

	key := make([]byte, Length)
	if _, err := rand.Read(key); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "generate the master key")
	}

	protected, err := dpapi.Protect(key)
	if err != nil {
		return nil, err
	}
	if err = os.WriteFile(path, protected, fileMode); err != nil {
		return nil, errors.Wrap(err, errors.Internal, "write the master key")
	}
	return key, nil
}
