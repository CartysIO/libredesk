package main

import (
	"testing"
	"time"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
)

func TestValidateHistoricalCreatedAt(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		conversationStart time.Time
		createdAt         time.Time
		wantErr           error
	}{
		{
			name:              "ok mid timeline",
			conversationStart: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			createdAt:         time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:              "ok equal conversation start",
			conversationStart: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			createdAt:         time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name:              "future beyond skew",
			conversationStart: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			createdAt:         now.Add(5 * time.Minute),
			wantErr:           errBackdatedCreatedAtFuture,
		},
		{
			name:              "before conversation",
			conversationStart: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			createdAt:         time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
			wantErr:           errBackdatedCreatedAtBeforeTicket,
		},
		{
			// After ticket open but older than the 10-year window (ticket opened long ago).
			name:              "too old",
			conversationStart: time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
			createdAt:         now.Add(-11 * 365 * 24 * time.Hour),
			wantErr:           errBackdatedCreatedAtTooOld,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHistoricalCreatedAt(tt.createdAt, tt.conversationStart, now)
			if err != tt.wantErr {
				t.Fatalf("validateHistoricalCreatedAt() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserCanWriteBackdated(t *testing.T) {
	yes := umodels.User{Permissions: pq.StringArray{authzmodels.PermMessagesWriteBackdated, authzmodels.PermMessagesWrite}}
	no := umodels.User{Permissions: pq.StringArray{authzmodels.PermMessagesWrite}}
	if !userCanWriteBackdated(yes) {
		t.Fatal("expected backdated permission")
	}
	if userCanWriteBackdated(no) {
		t.Fatal("unexpected backdated permission")
	}
}

func TestHistoricalPathRequiresBackdatedPermission(t *testing.T) {
	user := umodels.User{Permissions: pq.StringArray{authzmodels.PermMessagesWrite}}
	req := messageReq{SenderType: umodels.UserTypeAgent}
	if !canCreateConversationMessage(user, req) {
		t.Fatal("base write should pass")
	}
	if userCanWriteBackdated(user) {
		t.Fatal("should lack backdated perm")
	}
}
