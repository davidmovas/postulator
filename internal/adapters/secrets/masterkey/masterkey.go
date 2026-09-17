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

	tempSuffix    = ".tmp"
	directoryMode = 0o700
	fileMode      = 0o600

	unreadableKey = "master key cannot be unprotected"
)

type Config struct {
	Dir      string
	Recovery string
}

func (c Config) unreadable() string {
	if c.Recovery == "" {
		return unreadableKey
	}
	return unreadableKey + "; " + c.Recovery
}

func Load(cfg Config) ([]byte, error) {
	if cfg.Dir == "" {
		return nil, errors.New(errors.Invalid, "the master key directory must not be empty")
	}

	path := filepath.Join(cfg.Dir, FileName)
	protected, err := os.ReadFile(path)
	if stderrors.Is(err, fs.ErrNotExist) {
		return create(path)
	}
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "read the master key")
	}

	key, err := dpapi.Unprotect(protected)
	if err != nil {
		return nil, errors.New(errors.Locked, cfg.unreadable()).WithInternal(err)
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
	if err := writeAtomically(path, protected); err != nil {
		return nil, err
	}
	return key, nil
}

func writeAtomically(path string, contents []byte) error {
	temp := path + tempSuffix

	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "create the master key temporary file")
	}

	err = writeAndSync(file, contents)
	if closeErr := file.Close(); closeErr != nil && err == nil {
		err = errors.Wrap(closeErr, errors.Internal, "close the master key temporary file")
	}
	if err != nil {
		return stderrors.Join(err, remove(temp))
	}

	if err = os.Rename(temp, path); err != nil {
		return stderrors.Join(
			errors.Wrap(err, errors.Internal, "move the master key into place"),
			remove(temp),
		)
	}
	return nil
}

func writeAndSync(file *os.File, contents []byte) error {
	if _, err := file.Write(contents); err != nil {
		return errors.Wrap(err, errors.Internal, "write the master key")
	}
	return errors.Wrap(file.Sync(), errors.Internal, "flush the master key")
}

func remove(path string) error {
	if err := os.Remove(path); err != nil && !stderrors.Is(err, fs.ErrNotExist) {
		return errors.Wrap(err, errors.Internal, "remove the master key temporary file")
	}
	return nil
}
