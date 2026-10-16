package main

import (
	"testing"

	oidcmodels "github.com/abhinavxd/libredesk/internal/oidc/models"
)

func TestIsLocalLoginEnabled(t *testing.T) {
	tests := []struct {
		name       string
		configured *bool
		providers  []oidcmodels.OIDC
		clerk      bool
		want       bool
	}{
		{name: "defaults to enabled", want: true},
		{name: "remains enabled when configured", configured: boolPointer(true), want: true},
		{name: "remains enabled without an enabled OIDC provider", configured: boolPointer(false), providers: []oidcmodels.OIDC{{Enabled: false}}, want: true},
		{name: "is disabled with an enabled OIDC provider", configured: boolPointer(false), providers: []oidcmodels.OIDC{{Enabled: true}}, want: false},
		{name: "is disabled with Clerk and no OIDC provider", configured: boolPointer(false), clerk: true, want: false},
		{name: "stays enabled with Clerk when configured", configured: boolPointer(true), clerk: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ko.Delete(localLoginEnabledKey)
			if tt.configured != nil {
				if err := ko.Set(localLoginEnabledKey, *tt.configured); err != nil {
					t.Fatalf("set config: %v", err)
				}
			}
			t.Cleanup(func() { ko.Delete(localLoginEnabledKey) })

			if got := isLocalLoginEnabled(tt.providers, tt.clerk); got != tt.want {
				t.Fatalf("isLocalLoginEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func boolPointer(value bool) *bool {
	return &value
}
