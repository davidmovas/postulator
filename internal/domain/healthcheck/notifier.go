package healthcheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/logger"

	"github.com/gen2brain/beeep"
)

type siteNotificationState struct {
	wasDown      bool
	lastNotified time.Time
}

type notifier struct {
	states map[int64]*siteNotificationState
	mu     sync.RWMutex
	logger *logger.Logger
}

func NewNotifier(logger *logger.Logger) Notifier {
	return &notifier{
		states: make(map[int64]*siteNotificationState),
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

	// Фильтруем только те сайты, которые раньше не были down или давно не уведомляли
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

		// Уведомляем только если сайт упал впервые (не был down)
		if !state.wasDown {
			sitesToNotify = append(sitesToNotify, site)
			state.wasDown = true
			state.lastNotified = now
		}
	}

	if len(sitesToNotify) == 0 {
		return nil
	}

	// Формируем сообщение
	title := "Health Check Alert"
	message := n.formatUnhealthyMessage(sitesToNotify)

	n.logger.Warnf("Sending notification for %d unhealthy sites", len(sitesToNotify))

	if withSound {
		err := beeep.Alert(title, message, "")
		if err != nil {
			n.logger.ErrorWithErr(err, "Failed to send notification with sound")
			return err
		}
	} else {
		err := beeep.Notify(title, message, "")
		if err != nil {
			n.logger.ErrorWithErr(err, "Failed to send notification")
			return err
		}
	}

	return nil
}

func (n *notifier) NotifyRecoveredSites(ctx context.Context, sites []*entities.Site, withSound bool) error {
	if len(sites) == 0 {
		return nil
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Сбрасываем состояние для восстановленных сайтов
	for _, site := range sites {
		if state, exists := n.states[site.ID]; exists {
			state.wasDown = false
		}
	}

	// Формируем сообщение
	title := "Health Check Recovery"
	message := n.formatRecoveredMessage(sites)

	n.logger.Infof("Sending recovery notification for %d sites", len(sites))

	if withSound {
		err := beeep.Alert(title, message, "")
		if err != nil {
			n.logger.ErrorWithErr(err, "Failed to send recovery notification with sound")
			return err
		}
	} else {
		err := beeep.Notify(title, message, "")
		if err != nil {
			n.logger.ErrorWithErr(err, "Failed to send recovery notification")
			return err
		}
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
		return fmt.Sprintf("Site '%s' is unhealthy", sites[0].Name)
	}

	if len(sites) <= 3 {
		names := make([]string, len(sites))
		for i, site := range sites {
			names[i] = site.Name
		}
		return fmt.Sprintf("Sites are unhealthy: %v", names)
	}

	return fmt.Sprintf("%d sites are unhealthy. Check the application for details.", len(sites))
}

func (n *notifier) formatRecoveredMessage(sites []*entities.Site) string {
	if len(sites) == 1 {
		return fmt.Sprintf("Site '%s' has recovered", sites[0].Name)
	}

	if len(sites) <= 3 {
		names := make([]string, len(sites))
		for i, site := range sites {
			names[i] = site.Name
		}
		return fmt.Sprintf("Sites recovered: %v", names)
	}

	return fmt.Sprintf("%d sites have recovered", len(sites))
}
