package export

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"io"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	DatabaseName = "postulator.db"
	ManifestName = "manifest.json"
	Version      = 1

	workPrefix    = "postulator-backup"
	fileMode      = 0o600
	directoryMode = 0o700
)

type source interface {
	Snapshot(ctx context.Context, path string) error
}

type Manifest struct {
	Version   int    `json:"version"`
	CreatedAt string `json:"createdAt"`
	Database  string `json:"database"`
	Digest    string `json:"digest"`
	Bytes     int64  `json:"bytes"`
}

type Archive struct {
	source source
	clock  clock.Clock
}

func New(src source, now clock.Clock) *Archive {
	return &Archive{source: src, clock: now}
}

func (a *Archive) Write(ctx context.Context, path, password string) (err error) {
	if path == "" {
		return errors.New(errors.Invalid, "the backup path must not be empty")
	}
	if password == "" {
		return errors.New(errors.Invalid, "the backup password must not be empty")
	}

	work, err := os.MkdirTemp("", workPrefix)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "create the backup working directory")
	}
	defer func() {
		err = stderrors.Join(err, discard(work))
	}()

	snapshot := filepath.Join(work, DatabaseName)
	if err := a.source.Snapshot(ctx, snapshot); err != nil {
		return err
	}

	digest, size, err := fingerprint(snapshot)
	if err != nil {
		return err
	}

	manifest, err := json.Marshal(Manifest{
		Version:   Version,
		CreatedAt: a.clock.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Database:  DatabaseName,
		Digest:    digest,
		Bytes:     size,
	})
	if err != nil {
		return errors.Wrap(err, errors.Internal, "encode the backup manifest")
	}

	if err = os.MkdirAll(filepath.Dir(path), directoryMode); err != nil {
		return errors.Wrap(err, errors.Internal, "create the backup directory")
	}

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "create the backup file")
	}
	defer func() {
		err = stderrors.Join(err, closeFile(out, "close the backup file"))
		if err != nil {
			err = stderrors.Join(err, discard(path))
		}
	}()

	return seal(out, password, manifest, snapshot, size)
}

func seal(out io.Writer, password string, manifest []byte, snapshot string, size int64) error {
	frames, err := newSealer(out, password)
	if err != nil {
		return err
	}

	archive := tar.NewWriter(frames)
	if err := writeEntry(archive, ManifestName, int64(len(manifest)), nil, manifest); err != nil {
		return err
	}

	body, err := os.Open(snapshot)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "read the database snapshot")
	}

	err = writeEntry(archive, DatabaseName, size, body, nil)
	err = stderrors.Join(err, closeFile(body, "close the database snapshot"))
	if err != nil {
		return err
	}

	if err = archive.Close(); err != nil {
		return errors.Wrap(err, errors.Internal, "close the backup archive")
	}
	return frames.Close()
}

func writeEntry(archive *tar.Writer, name string, size int64, body io.Reader, inline []byte) error {
	if err := archive.WriteHeader(&tar.Header{
		Name: name, Mode: fileMode, Size: size, Typeflag: tar.TypeReg,
	}); err != nil {
		return errors.Wrap(err, errors.Internal, "write the header of "+name)
	}

	var err error
	if body != nil {
		_, err = io.Copy(archive, body)
	} else {
		_, err = archive.Write(inline)
	}
	return errors.Wrap(err, errors.Internal, "write "+name+" into the backup")
}

func (a *Archive) Read(_ context.Context, path, password string) (dir string, err error) {
	if path == "" {
		return "", errors.New(errors.Invalid, "the backup path must not be empty")
	}

	in, err := os.Open(path)
	if err != nil {
		return "", errors.Wrap(err, errors.NotFound, "read the backup file")
	}
	defer func() {
		err = stderrors.Join(err, closeFile(in, "close the backup file"))
	}()

	dir, err = os.MkdirTemp("", workPrefix)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "create the restore directory")
	}
	defer func() {
		if err != nil {
			err = stderrors.Join(err, discard(dir))
			dir = ""
		}
	}()

	frames, err := newOpener(in, password)
	if err != nil {
		return "", err
	}
	if err := extract(tar.NewReader(frames), dir); err != nil {
		return "", err
	}
	if err := frames.complete(); err != nil {
		return "", err
	}
	return dir, verify(dir)
}

func extract(archive *tar.Reader, dir string) error {
	for {
		header, err := archive.Next()
		if stderrors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return errors.New(errors.Invalid, "the backup archive is not readable").WithInternal(err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != header.Name || header.Name == "." {
			return errors.New(errors.Invalid, "the backup archive carries an unexpected entry").
				WithDetail("name", header.Name)
		}

		if err := spill(archive, filepath.Join(dir, header.Name)); err != nil {
			return err
		}
	}
}

func spill(archive *tar.Reader, path string) (err error) {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return errors.Wrap(err, errors.Internal, "create "+filepath.Base(path))
	}
	defer func() {
		err = stderrors.Join(err, closeFile(out, "close "+filepath.Base(path)))
	}()

	if _, err = io.Copy(out, archive); err != nil {
		return errors.New(errors.Invalid, "the backup archive ends inside "+filepath.Base(path)).WithInternal(err)
	}
	return nil
}

func verify(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, ManifestName))
	if err != nil {
		return errors.New(errors.Invalid, "the backup carries no manifest").WithInternal(err)
	}

	var manifest Manifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return errors.New(errors.Invalid, "the backup manifest is not readable").WithInternal(err)
	}
	if manifest.Version != Version {
		return errors.New(errors.Invalid, "the backup was written by another version of Postulator").
			WithDetail("version", manifest.Version)
	}
	if manifest.Database != DatabaseName {
		return errors.New(errors.Invalid, "the backup manifest names another database").
			WithDetail("database", manifest.Database)
	}

	digest, size, err := fingerprint(filepath.Join(dir, manifest.Database))
	if err != nil {
		return err
	}
	if digest != manifest.Digest || size != manifest.Bytes {
		return errors.New(errors.Invalid, "the backup database does not match its manifest")
	}
	return nil
}

func fingerprint(path string) (digest string, size int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, errors.New(errors.Invalid, "read "+filepath.Base(path)).WithInternal(err)
	}
	defer func() {
		err = stderrors.Join(err, closeFile(file, "close "+filepath.Base(path)))
	}()

	sum := sha256.New()
	size, err = io.Copy(sum, file)
	if err != nil {
		return "", 0, errors.Wrap(err, errors.Internal, "hash "+filepath.Base(path))
	}
	return hex.EncodeToString(sum.Sum(nil)), size, nil
}

func closeFile(file *os.File, message string) error {
	return errors.Wrap(file.Close(), errors.Internal, message)
}

func discard(path string) error {
	return errors.Wrap(os.RemoveAll(path), errors.Internal, "remove "+filepath.Base(path))
}
