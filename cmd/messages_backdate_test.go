package main

import (
	"testing"
	"time"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
)

func TestValidateHistoricalCreatedAt(t *testing.T) {
	convCreated := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		createdAt time.Time
		wantErr   error
	}{
		{"ok mid timeline", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), nil},
		{"ok equal conversation start", convCreated, nil},
		{"future beyond skew", now.Add(5 * time.Minute), errBackdatedCreatedAtFuture},
		{"before conversation", convCreated.Add(-time.Hour), errBackdatedCreatedAtBeforeTicket},
		{"too old", now.Add(-11 * 365 * 24 * time.Hour), errBackdatedCreatedAtTooOld},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHistoricalCreatedAt(tt.createdAt, convCreated, now)
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
