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

func locked() error {
	return errors.New(errors.Locked, "the application is locked")
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Store == nil {
		return nil
	}

	protected, err := masterkey.Protected(c.keyConfig())
	if err != nil {
		return err
	}
	if !protected {
		return errors.New(errors.Invalid, "the application cannot be locked without a master password")
	}
	if err := c.shutdown(); err != nil {
		return err
	}
	return c.Events.Publish(events.AppLocked, events.AppLockedPayload{})
}

func (c *Core) Unlock(ctx context.Context, password string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Store != nil {
		return nil
	}

	key, err := masterkey.Unlock(c.keyConfig(), password)
	if err != nil {
		return err
	}
	if err := c.compose(ctx, key); err != nil {
		return err
	}
	return c.Events.Publish(events.AppUnlocked, events.AppUnlockedPayload{})
}

func (c *Core) SetMasterPassword(ctx context.Context, current, next string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

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

	if c.Store != nil {
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
	if c.key != nil {
		return c.key, nil
	}
	return masterkey.Load(c.keyConfig())
}

func (c *Core) ExportBackup(ctx context.Context, path, password string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Store == nil {
		return 0, locked()
	}
	if err := c.Archive.Write(ctx, path, password); err != nil {
		return 0, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrap(err, errors.Internal, "measure the backup file")
	}
	return info.Size(), nil
}

func (c *Core) ImportBackup(ctx context.Context, path, password string) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Store == nil {
		return locked()
	}

	restored, err := c.Archive.Read(ctx, path, password)
	if err != nil {
		return err
	}
	defer func() {
		err = stderrors.Join(err, errors.Wrap(os.RemoveAll(restored), errors.Internal, "remove the restored files"))
	}()

	key, store := c.key, c.Store
	c.quiesce()
	if err = store.Restore(ctx, filepath.Join(restored, export.DatabaseName)); err != nil {
		return stderrors.Join(err, c.abandon(store))
	}
	if err := c.abandon(store); err != nil {
		return err
	}
	return c.compose(ctx, key)
}

func (c *Core) quiesce() {
	if c.Scheduler != nil {
		c.Scheduler.Stop()
	}
	if c.Agent != nil {
		c.Agent.Close()
	}
	if c.Engine != nil {
		c.Engine.Stop()
	}
}

func (c *Core) abandon(store *sqlite.Store) error {
	c.kit = kit{}
	return store.Close()
}

func (c *Core) shutdown() error {
	store := c.Store
	if store == nil {
		return nil
	}

	c.quiesce()
	clear(c.key)
	c.key = nil
	return c.abandon(store)
}
