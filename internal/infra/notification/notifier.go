package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/gen2brain/beeep"
)

type Notifier interface {
	Notify(ctx context.Context, opts *Options) error
	NotifySimple(ctx context.Context, title, message string) error
	NotifyWithSound(ctx context.Context, title, message string) error
	SetDefaultIcon(iconPath string)
	SetDefaultIconBytes(iconBytes []byte)
	SetAppName(name string)
}

type Options struct {
	Title     string
	Message   string
	Icon      any
	WithSound bool
}

type Builder struct {
	opts     *Options
	notifier Notifier
}

type notifier struct {
	defaultIcon any
	appName     string
}

func New() Notifier {
	return &notifier{
		appName: "Postulator",
	}
}

func NewWithConfig(appName string, defaultIcon any) Notifier {
	beeep.AppName = appName
	return &notifier{
		appName:     appName,
		defaultIcon: defaultIcon,
	}
}

func (n *notifier) SetAppName(name string) {
	n.appName = name
	beeep.AppName = name
}

func (n *notifier) SetDefaultIcon(iconPath string) {
	n.defaultIcon = iconPath
}

func (n *notifier) SetDefaultIconBytes(iconBytes []byte) {
	n.defaultIcon = iconBytes
}

func (n *notifier) Notify(_ context.Context, opts *Options) error {
	if opts == nil {
		return fmt.Errorf("notification options cannot be nil")
	}

	if opts.Title == "" {
		return fmt.Errorf("notification title cannot be empty")
	}

	icon := opts.Icon
	if icon == nil {
		icon = n.defaultIcon
	}

	if opts.WithSound {
		return beeep.Alert(opts.Title, opts.Message, icon)
	}

	return beeep.Notify(opts.Title, opts.Message, icon)
}

func (n *notifier) NotifySimple(ctx context.Context, title, message string) error {
	return n.Notify(ctx, &Options{
		Title:   title,
		Message: message,
	})
}

func (n *notifier) NotifyWithSound(ctx context.Context, title, message string) error {
	return n.Notify(ctx, &Options{
		Title:     title,
		Message:   message,
		WithSound: true,
	})
}

func (n *notifier) Builder() *Builder {
	return &Builder{
		opts:     &Options{},
		notifier: n,
	}
}

func (b *Builder) Title(title string) *Builder {
	b.opts.Title = title
	return b
}

func (b *Builder) Message(message string) *Builder {
	b.opts.Message = message
	return b
}

func (b *Builder) Icon(icon any) *Builder {
	b.opts.Icon = icon
	return b
}

func (b *Builder) WithSound() *Builder {
	b.opts.WithSound = true
	return b
}

func (b *Builder) Send(ctx context.Context) error {
	return b.notifier.Notify(ctx, b.opts)
}

func NotifyError(ctx context.Context, n Notifier, err error) error {
	return n.NotifyWithSound(ctx, "Error", err.Error())
}

func NotifySuccess(ctx context.Context, n Notifier, message string) error {
	return n.NotifySimple(ctx, "Success", message)
}

func NotifyWarning(ctx context.Context, n Notifier, message string) error {
	return n.NotifyWithSound(ctx, "Warning", message)
}

func NotifyInfo(ctx context.Context, n Notifier, message string) error {
	return n.NotifySimple(ctx, "Info", message)
}

type BatchNotifier struct {
	notifier Notifier
	delay    time.Duration
}

func NewBatch(n Notifier, delay time.Duration) *BatchNotifier {
	return &BatchNotifier{
		notifier: n,
		delay:    delay,
	}
}

func (bn *BatchNotifier) Send(ctx context.Context, notifications []*Options) error {
	for i, opts := range notifications {
		if err := bn.notifier.Notify(ctx, opts); err != nil {
			return fmt.Errorf("failed to send notification %d: %w", i, err)
		}

		if i < len(notifications)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(bn.delay):
			}
		}
	}
	return nil
}
