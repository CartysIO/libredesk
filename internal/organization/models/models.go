// Package models holds the organization types.
package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// Organization is a customer company that contacts belong to.
type Organization struct {
	ID           int         `db:"id" json:"id"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at" json:"updated_at"`
	Name         string      `db:"name" json:"name"`
	ExternalID   null.String `db:"external_id" json:"external_id"`
	ContactCount int         `db:"contact_count" json:"contact_count"`
}

// Contact is a contact listed under an organization.
type Contact struct {
	ID             int         `db:"id" json:"id"`
	CreatedAt      time.Time   `db:"created_at" json:"created_at"`
	FirstName      string      `db:"first_name" json:"first_name"`
	LastName       string      `db:"last_name" json:"last_name"`
	Email          null.String `db:"email" json:"email"`
	AvatarURL      null.String `db:"avatar_url" json:"avatar_url"`
	ExternalUserID null.String `db:"external_user_id" json:"external_user_id"`
}
