package healthcheck

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/logger"

	"github.com/gen2brain/beeep"
)

//go:embed assets/icon.png
var icon []byte

const (
	unhealthyIntensificationAmount = 3
	recoveredIntensificationAmount = 3

	unhealthNotificationTitle  = "Health Check Alert"
	recoveredNotificationTitle = "Health Check Recovery"
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
	beeep.AppName = "Postulator"
	return &notifier{
		states: make(map[int64]*siteNotificationState),
		logger: logger.
			WithScope("notifier").
			WithScope("healthcheck"),
	}
}

func (n *notifier) NotifyUnhealthySites(_ context.Context, sites []*entities.Site, withSound bool) error {
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

	if withSound {
		err := beeep.Alert(unhealthNotificationTitle, message, icon)
		if err != nil {
			n.logger.ErrorWithErr(err, "Failed to send notification with sound")
			return err
		}
	} else {
		err := beeep.Notify(unhealthNotificationTitle, message, icon)
		if err != nil {
			n.logger.ErrorWithErr(err, "Failed to send notification")
			return err
		}
	}

	return nil
}

func (n *notifier) NotifyRecoveredSites(_ context.Context, sites []*entities.Site, withSound bool) error {
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

	if withSound {
		err := beeep.Alert(recoveredNotificationTitle, message, "")
		if err != nil {
			n.logger.ErrorWithErr(err, "Failed to send recovery notification with sound")
			return err
		}
	} else {
		err := beeep.Notify(recoveredNotificationTitle, message, "")
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
		site := sites[0]
		return fmt.Sprintf("Site Down\n\n%s", site.Name)
	}

	if len(sites) <= unhealthyIntensificationAmount {
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
	for i := 0; i < unhealthyIntensificationAmount; i++ {
		msg.WriteString(fmt.Sprintf("%s\n", sites[i].Name))
	}

	msg.WriteString(fmt.Sprintf("...and %d more sites", len(sites)-unhealthyIntensificationAmount))
	return msg.String()
}

func (n *notifier) formatRecoveredMessage(sites []*entities.Site) string {
	if len(sites) == 1 {
		site := sites[0]
		return fmt.Sprintf("Site Recovered\n\n%s", site.Name)
	}

	if len(sites) <= recoveredIntensificationAmount {
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
	for i := 0; i < recoveredIntensificationAmount; i++ {
		msg.WriteString(fmt.Sprintf("%s\n", sites[i].Name))
	}

	msg.WriteString(fmt.Sprintf("...and %d more sites", len(sites)-recoveredIntensificationAmount))
	return msg.String()
}
