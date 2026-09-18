package masterkey

import (
	stderrors "errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/secrets/dpapi"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterpassword"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const ProtectedFileName = FileName + ".pw"

func Protected(cfg Config) (bool, error) {
	if cfg.Dir == "" {
		return false, errors.New(errors.Invalid, "the master key directory must not be empty")
	}

	_, err := os.Stat(filepath.Join(cfg.Dir, ProtectedFileName))
	if stderrors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, errors.Internal, "look for the wrapped master key")
	}
	return true, nil
}

func Unlock(cfg Config, password string) ([]byte, error) {
	if cfg.Dir == "" {
		return nil, errors.New(errors.Invalid, "the master key directory must not be empty")
	}

	envelope, err := os.ReadFile(filepath.Join(cfg.Dir, ProtectedFileName))
	if stderrors.Is(err, fs.ErrNotExist) {
		return nil, errors.New(errors.NotFound, "no master password is set")
	}
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "read the wrapped master key")
	}

	protected, err := masterpassword.Open(password, envelope)
	if err != nil {
		return nil, err
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

func SetPassword(cfg Config, key []byte, password string) error {
	if cfg.Dir == "" {
		return errors.New(errors.Invalid, "the master key directory must not be empty")
	}
	if len(key) != Length {
		return errors.New(errors.Invalid, "the master key has the wrong length")
	}

	protected, err := dpapi.Protect(key)
	if err != nil {
		return err
	}

	envelope, err := masterpassword.Seal(password, protected)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(cfg.Dir, directoryMode); err != nil {
		return errors.Wrap(err, errors.Internal, "create the master key directory")
	}
	if err := writeAtomically(filepath.Join(cfg.Dir, ProtectedFileName), envelope); err != nil {
		return err
	}
	return discard(filepath.Join(cfg.Dir, FileName))
}

func ClearPassword(cfg Config, key []byte) error {
	if cfg.Dir == "" {
		return errors.New(errors.Invalid, "the master key directory must not be empty")
	}
	if len(key) != Length {
		return errors.New(errors.Invalid, "the master key has the wrong length")
	}

	protected, err := dpapi.Protect(key)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(cfg.Dir, directoryMode); err != nil {
		return errors.Wrap(err, errors.Internal, "create the master key directory")
	}
	if err := writeAtomically(filepath.Join(cfg.Dir, FileName), protected); err != nil {
		return err
	}
	return discard(filepath.Join(cfg.Dir, ProtectedFileName))
}

func discard(path string) error {
	if err := os.Remove(path); err != nil && !stderrors.Is(err, fs.ErrNotExist) {
		return errors.Wrap(err, errors.Internal, "remove the superseded master key file")
	}
	return nil
}
