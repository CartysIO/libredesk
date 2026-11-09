package main

import (
	"errors"
	"slices"
	"time"

	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

const (
	backdateFutureSkew = 2 * time.Minute
	backdateMaxAge     = 10 * 365 * 24 * time.Hour
)

var (
	errBackdatedCreatedAtFuture          = errors.New("created_at must not be in the future")
	errBackdatedCreatedAtBeforeTicket    = errors.New("created_at must not be before the conversation was created")
	errBackdatedCreatedAtTooOld          = errors.New("created_at is too far in the past")
	errBackdatedCreatedAtAgentOnly       = errors.New("created_at is only supported for agent messages")
	errBackdatedCreatedAtNoWhatsAppExtra = errors.New("created_at cannot be used with WhatsApp template fields")
)

func userCanWriteBackdated(user umodels.User) bool {
	return slices.Contains(user.Permissions, authzModels.PermMessagesWriteBackdated)
}

func validateHistoricalCreatedAt(createdAt, conversationCreatedAt, now time.Time) error {
	if createdAt.After(now.Add(backdateFutureSkew)) {
		return errBackdatedCreatedAtFuture
	}
	if createdAt.Before(conversationCreatedAt) {
		return errBackdatedCreatedAtBeforeTicket
	}
	if createdAt.Before(now.Add(-backdateMaxAge)) {
		return errBackdatedCreatedAtTooOld
	}
	return nil
}
