package conversation

import (
	"context"
	"fmt"
	"time"
)

// RunUnsnoozer runs the conversation unsnoozer.
func (c *Manager) RunUnsnoozer(ctx context.Context, unsnoozeInterval time.Duration) {
	ticker := time.NewTicker(unsnoozeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.unsnoozeAll(ctx)
		}
	}
}

// unsnoozeAll unsnoozes all expired snoozed conversations. Each goes to "On you" if an agent already
// replied, else "Open"; the broadcast and the status change activity use the status it actually got.
func (c *Manager) unsnoozeAll(ctx context.Context) {
	rows, err := c.q.UnsnoozeAll.QueryxContext(ctx)
	if err != nil {
		c.lo.Error("error unsnoozing all conversations", "error", err)
		return
	}
	type unsnoozed struct {
		UUID   string `db:"uuid"`
		Status string `db:"status"`
	}
	var done []unsnoozed
	for rows.Next() {
		var u unsnoozed
		if err := rows.StructScan(&u); err != nil {
			c.lo.Error("error reading unsnoozed conversation", "error", err)
			continue
		}
		done = append(done, u)
	}
	if err := rows.Err(); err != nil {
		c.lo.Error("error reading unsnoozed conversations", "error", err)
	}
	rows.Close()
	if len(done) == 0 {
		return
	}
	c.lo.Info(fmt.Sprintf("unsnoozed %d conversations", len(done)))

	systemUser, err := c.userStore.GetSystemUser()
	if err != nil {
		c.lo.Error("error fetching system user after unsnooze", "error", err)
	}
	for _, u := range done {
		c.BroadcastConversationUpdate(u.UUID, map[string]any{"status": u.Status, "snoozed_until": nil})
		if err == nil {
			if recErr := c.RecordStatusChange(u.Status, u.UUID, systemUser); recErr != nil {
				c.lo.Error("error recording status change after unsnooze", "uuid", u.UUID, "error", recErr)
			}
		}
	}
}
