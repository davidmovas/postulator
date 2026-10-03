package openai

import "time"

type Alarm = alarm

func WithAlarm(arm func(after time.Duration, ring func()) Alarm) Option {
	return func(c *Client) {
		if arm != nil {
			c.arm = arm
		}
	}
}
