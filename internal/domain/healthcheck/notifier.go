package healthcheck

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/notification"
	"github.com/davidmovas/postulator/internal/infra/notifyicon"
	"github.com/davidmovas/postulator/pkg/logger"
)

const (
	unhealthNotificationTitle  = "Health Check Alert"
	recoveredNotificationTitle = "Health Check Recovery"
	maxDisplay                 = 2
)

type siteNotificationState struct {
	wasDown      bool
	lastNotified time.Time
}

type notifier struct {
	baseNotifier notification.Notifier
	states       map[int64]*siteNotificationState
	mu           sync.RWMutex
	logger       *logger.Logger
}

func NewNotifier(logger *logger.Logger) Notifier {
	iconBytes := notifyicon.Icon()

	return &notifier{
		baseNotifier: notification.NewWithConfig("Postulator", iconBytes),
		states:       make(map[int64]*siteNotificationState),
		logger: logger.
			WithScope("notifier").
			WithScope("healthcheck"),
	}
}

func (n *notifier) NotifyUnhealthySites(ctx context.Context, sites []*entities.Site, withSound bool) error {
	if len(sites) == 0 {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	var sitesToNotify []*entities.Site
	now := time.Now()

	for _, site := range sites {
		state, exists := n.states[site.ID]
		if !exists {
			state = &siteNotificationState{
				wasDown:      false,
				lastNotified: time.Time{},
			}
			n.states[site.ID] = state
		}

		if !state.wasDown {
			sitesToNotify = append(sitesToNotify, site)
			state.wasDown = true
			state.lastNotified = now
		}
	}

	if len(sitesToNotify) == 0 {
		return nil
	}

	message := n.formatUnhealthyMessage(sitesToNotify)

	opts := &notification.Options{
		Title:     unhealthNotificationTitle,
		Message:   message,
		WithSound: withSound,
	}

	if err := n.baseNotifier.Notify(ctx, opts); err != nil {
		n.logger.ErrorWithErr(err, "Failed to send notification")
		return err
	}

	return nil
}

func (n *notifier) NotifyRecoveredSites(ctx context.Context, sites []*entities.Site, withSound bool) error {
	if len(sites) == 0 {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	for _, site := range sites {
		if state, exists := n.states[site.ID]; exists {
			state.wasDown = false
		}
	}

	message := n.formatRecoveredMessage(sites)

	opts := &notification.Options{
		Title:     recoveredNotificationTitle,
		Message:   message,
		WithSound: withSound,
	}

	if err := n.baseNotifier.Notify(ctx, opts); err != nil {
		n.logger.ErrorWithErr(err, "Failed to send recovery notification")
		return err
	}

	return nil
}

func (n *notifier) ResetState(siteID int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.states, siteID)
}

func (n *notifier) formatUnhealthyMessage(sites []*entities.Site) string {
	if len(sites) == 1 {
		site := sites[0]
		return fmt.Sprintf("Site Down\n\n%s", site.Name)
	}

	if len(sites) <= maxDisplay {
		var msg strings.Builder
		msg.WriteString(fmt.Sprintf("%d Sites Down\n\n", len(sites)))
		for i, site := range sites {
			msg.WriteString(site.Name)
			if i < len(sites)-1 {
				msg.WriteString("\n")
			}
		}
		return msg.String()
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("%d Sites Down\n\n", len(sites)))
	for i := 0; i < maxDisplay; i++ {
		msg.WriteString(fmt.Sprintf("%s\n", sites[i].Name))
	}
	msg.WriteString(fmt.Sprintf("and %d more", len(sites)-maxDisplay))
	return msg.String()
}

func (n *notifier) formatRecoveredMessage(sites []*entities.Site) string {
	if len(sites) == 1 {
		site := sites[0]
		return fmt.Sprintf("Site Recovered\n\n%s", site.Name)
	}

	if len(sites) <= maxDisplay {
		var msg strings.Builder
		msg.WriteString(fmt.Sprintf("%d Sites Recovered\n\n", len(sites)))
		for i, site := range sites {
			msg.WriteString(site.Name)
			if i < len(sites)-1 {
				msg.WriteString("\n")
			}
		}
		return msg.String()
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("%d Sites Recovered\n\n", len(sites)))
	for i := 0; i < maxDisplay; i++ {
		msg.WriteString(fmt.Sprintf("%s\n", sites[i].Name))
	}
	msg.WriteString(fmt.Sprintf("and %d more", len(sites)-maxDisplay))
	return msg.String()
}
