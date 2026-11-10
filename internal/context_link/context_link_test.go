package contextlink

import (
	"encoding/json"
	"testing"

	authmodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/context_link/models"
	convmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestGenerateURLOrganizationVariables(t *testing.T) {
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo}
	link := models.ContextLink{URLTemplate: "https://admin.example.com/teams/{{organization_external_id}}?org={{organization_id}}&name={{organization_name}}" +
		"&stripe={{organization_custom_attributes.stripe_customer_id}}&seats={{organization_custom_attributes.seats}}" +
		"&missing={{organization_custom_attributes.nope}}&email={{email}}"}

	conv := convmodels.Conversation{
		UUID:                   "c1",
		OrganizationID:         null.IntFrom(7),
		OrganizationName:       null.StringFrom("Acme & Co"),
		OrganizationExternalID: null.StringFrom("team_1"),
		OrganizationAttributes: json.RawMessage(`{"stripe_customer_id":"cus_123","seats":15}`),
		Contact:                convmodels.ConversationContact{Email: null.StringFrom("a@acme.com")},
	}
	got, err := m.GenerateURL(link, conv, authmodels.User{})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://admin.example.com/teams/team_1?org=7&name=Acme+%26+Co&stripe=cus_123&seats=15&missing=&email=a%40acme.com"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}

	// A conversation without an organization gets empty values.
	got, err = m.GenerateURL(link, convmodels.Conversation{UUID: "c2", OrganizationAttributes: json.RawMessage(`null`)}, authmodels.User{})
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://admin.example.com/teams/?org=&name=&stripe=&seats=&missing=&email="; got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}
