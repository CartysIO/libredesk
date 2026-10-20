// Package models holds the organization types.
package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"

	"github.com/volatiletech/null/v9"
)

// Organization is a customer company that contacts belong to.
type Organization struct {
	ID         int         `db:"id" json:"id"`
	CreatedAt  time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time   `db:"updated_at" json:"updated_at"`
	Name       string      `db:"name" json:"name"`
	ExternalID null.String `db:"external_id" json:"external_id"`
	// CustomAttributes holds the values of the custom attributes that apply to organizations.
	CustomAttributes json.RawMessage `db:"custom_attributes" json:"custom_attributes"`
	// Domains are lowercase email domains. An email contact with no organization joins the one organization
	// that has its domain.
	Domains      pq.StringArray `db:"domains" json:"domains"`
	ContactCount int            `db:"contact_count" json:"contact_count"`
}

// Fields are the organization fields set apart from the name and external ID. A nil field is left unchanged.
type Fields struct {
	Domains          []string       `json:"domains"`
	CustomAttributes map[string]any `json:"custom_attributes"`
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
	// IsDefault is true when this organization is the contact's default organization.
	IsDefault bool `db:"is_default" json:"is_default"`
}

// ContactOrganization is an organization that a contact belongs to.
type ContactOrganization struct {
	ID         int         `db:"id" json:"id"`
	Name       string      `db:"name" json:"name"`
	ExternalID null.String `db:"external_id" json:"external_id"`
	IsDefault  bool        `db:"is_default" json:"is_default"`
}

// MembershipChange tells what adding or removing a membership changed.
type MembershipChange struct {
	Added          int `db:"added"`
	DefaultChanged int `db:"default_changed"`
	// Filed lists the conversations that had no organization and got the contact's default.
	Filed pq.StringArray `db:"filed"`
}

// DefaultChange is a contact whose default organization changed.
type DefaultChange struct {
	ContactID      int      `db:"id"`
	OrganizationID null.Int `db:"organization_id"`
}
