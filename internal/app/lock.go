package app

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/secrets/export"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const rollbackPrefix = "postulator-import"

func locked() error {
	return errors.New(errors.Locked, "the application is locked")
}

func transitioning() error {
	return errors.New(errors.Conflict, "the application is already locking or unlocking")
}

func (c *Core) Locked() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Store == nil
}

func (c *Core) Protected() (bool, error) {
	return masterkey.Protected(c.keyConfig())
}

func (c *Core) Lock() error {
	if !c.changing.CompareAndSwap(false, true) {
		return transitioning()
	}
	defer c.changing.Store(false)

	if c.Locked() {
		return nil
	}

	protected, err := masterkey.Protected(c.keyConfig())
	if err != nil {
		return err
	}
	if !protected {
		return errors.New(errors.Invalid, "the application cannot be locked without a master password")
	}

	retired, key := c.detach()
	if retired.Store == nil {
		return nil
	}

	quiesce(retired)
	clear(key)
	if closeErr := retired.Store.Close(); closeErr != nil {
		return closeErr
	}
	return c.Events.Publish(events.AppLocked, events.AppLockedPayload{})
}

func (c *Core) Unlock(ctx context.Context, password string) error {
	if !c.changing.CompareAndSwap(false, true) {
		return transitioning()
	}
	defer c.changing.Store(false)

	if !c.Locked() {
		return nil
	}

	key, err := masterkey.Unlock(c.keyConfig(), password)
	if err != nil {
		return err
	}
	if composeErr := c.compose(ctx, key); composeErr != nil {
		return composeErr
	}
	return c.Events.Publish(events.AppUnlocked, events.AppUnlockedPayload{})
}

func (c *Core) SetMasterPassword(ctx context.Context, current, next string) error {
	if !c.changing.CompareAndSwap(false, true) {
		return transitioning()
	}
	defer c.changing.Store(false)

	key, err := c.currentKey(current)
	if err != nil {
		return err
	}

	if next == "" {
		err = masterkey.ClearPassword(c.keyConfig(), key)
	} else {
		err = masterkey.SetPassword(c.keyConfig(), key, next)
	}
	if err != nil {
		return err
	}

	if !c.Locked() {
		return nil
	}
	if composeErr := c.compose(ctx, key); composeErr != nil {
		return composeErr
	}
	return c.Events.Publish(events.AppUnlocked, events.AppUnlockedPayload{})
}

func (c *Core) currentKey(current string) ([]byte, error) {
	protected, err := masterkey.Protected(c.keyConfig())
	if err != nil {
		return nil, err
	}
	if protected {
		return masterkey.Unlock(c.keyConfig(), current)
	}

	c.mu.RLock()
	held := c.key
	c.mu.RUnlock()

	if held != nil {
		return held, nil
	}
	return masterkey.Load(c.keyConfig())
}

func (c *Core) ExportBackup(ctx context.Context, path, password string) (int64, error) {
	c.mu.RLock()
	archive := c.Archive
	c.mu.RUnlock()

	if archive == nil {
		return 0, locked()
	}
	if err := archive.Write(ctx, path, password); err != nil {
		return 0, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrap(err, errors.Internal, "measure the backup file")
	}
	return info.Size(), nil
}

func (c *Core) ImportBackup(ctx context.Context, path, password string) (err error) {
	if !c.changing.CompareAndSwap(false, true) {
		return transitioning()
	}
	defer c.changing.Store(false)

	c.mu.RLock()
	archive := c.Archive
	c.mu.RUnlock()

	if archive == nil {
		return locked()
	}

	restored, err := archive.Read(ctx, path, password)
	if err != nil {
		return err
	}
	defer func() {
		err = stderrors.Join(err, errors.Wrap(os.RemoveAll(restored), errors.Internal, "remove the restored files"))
	}()

	retired, key := c.detach()
	if retired.Store == nil {
		return locked()
	}
	quiesce(retired)

	kept, err := c.keep(ctx, retired.Store)
	if err != nil {
		return stderrors.Join(err, retired.Store.Close(), c.compose(ctx, key))
	}

	if swapErr := c.swap(ctx, retired.Store, filepath.Join(restored, export.DatabaseName), key); swapErr != nil {
		stranded, rolled := c.rollBack(ctx, key, filepath.Join(kept, export.DatabaseName), swapErr)
		if stranded {
			return rolled
		}
		return stderrors.Join(rolled, discard(kept))
	}
	return discard(kept)
}

func (c *Core) keep(ctx context.Context, store *sqlite.Store) (string, error) {
	work, err := os.MkdirTemp("", rollbackPrefix)
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "create the working directory of the import")
	}
	if snapshotErr := store.Snapshot(ctx, filepath.Join(work, export.DatabaseName)); snapshotErr != nil {
		return "", stderrors.Join(snapshotErr, discard(work))
	}
	return work, nil
}

func (c *Core) swap(ctx context.Context, store *sqlite.Store, from string, key []byte) error {
	if err := store.Restore(ctx, from); err != nil {
		return stderrors.Join(err, store.Close())
	}
	if err := store.Close(); err != nil {
		return err
	}
	return c.compose(ctx, key)
}

func (c *Core) rollBack(ctx context.Context, key []byte, kept string, cause error) (stranded bool, err error) {
	store, openErr := c.openStore(key)
	if openErr != nil {
		return true, stderrors.Join(cause, c.abandoned(kept, openErr))
	}
	if restoreErr := store.Restore(ctx, kept); restoreErr != nil {
		return true, stderrors.Join(cause, c.abandoned(kept, restoreErr), store.Close())
	}
	if closeErr := store.Close(); closeErr != nil {
		return true, stderrors.Join(cause, c.abandoned(kept, closeErr))
	}
	if composeErr := c.compose(ctx, key); composeErr != nil {
		return true, stderrors.Join(cause, c.abandoned(kept, composeErr))
	}
	return false, cause
}

func (c *Core) abandoned(kept string, cause error) error {
	return errors.Wrap(cause, errors.Internal,
		"the import failed and the application stayed locked; the database it held before the import is kept at "+
			kept+" and can be imported once the application starts again")
}

func discard(work string) error {
	return errors.Wrap(os.RemoveAll(work), errors.Internal, "remove the working directory of the import")
}

func quiesce(retired kit) {
	if retired.Scheduler != nil {
		retired.Scheduler.Stop()
	}
	if retired.Agent != nil {
		retired.Agent.Close()
	}
	if retired.Engine != nil {
		retired.Engine.Stop()
	}
}

func (c *Core) detach() (retired kit, key []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	retired, key = c.kit, c.key
	c.kit = kit{}
	c.key = nil
	return retired, key
}

func (c *Core) install(key []byte, built kit) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}
	c.key = key
	c.kit = built
	return true
}
