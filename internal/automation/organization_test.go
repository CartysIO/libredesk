package automation

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/volatiletech/null/v9"
)

func TestOrganizationConditions(t *testing.T) {
	e := createTestEngine(&mockConversationStore{})
	top := createTestConversation(func(c *cmodels.Conversation) {
		c.OrganizationID = null.IntFrom(12)
		c.OrganizationAttributes = json.RawMessage(`{"tier":"Top","seats":15,"vip":true}`)
	})
	none := createTestConversation()
	none.OrganizationAttributes = json.RawMessage(`null`)

	org := func(op, value string) models.RuleDetail {
		return models.RuleDetail{Field: models.ConversationOrganization, FieldType: models.FieldTypeConversationField, Operator: op, Value: value}
	}
	attr := func(field, op, value string) models.RuleDetail {
		return models.RuleDetail{Field: field, FieldType: models.FieldTypeOrganizationCustomAttribute, Operator: op, Value: value}
	}
	cases := []struct {
		name string
		rule models.RuleDetail
		conv cmodels.Conversation
		want bool
	}{
		{"equals", org(models.RuleOperatorEquals, "12"), top, true},
		{"equals other", org(models.RuleOperatorEquals, "1"), top, false},
		{"not equals", org(models.RuleOperatorNotEqual, "1"), top, true},
		{"is any of", org(models.RuleOperatorContains, "3, 12,40"), top, true},
		{"is any of matches whole ids only", org(models.RuleOperatorContains, "1,2"), top, false},
		{"is none of", org(models.RuleOperatorNotContains, "1,2"), top, true},
		{"is none of hit", org(models.RuleOperatorNotContains, "12"), top, false},
		{"no org is not any of", org(models.RuleOperatorContains, "12"), none, false},
		{"no org is none of", org(models.RuleOperatorNotContains, "12"), none, true},
		{"set", org(models.RuleOperatorSet, ""), top, true},
		{"not set", org(models.RuleOperatorNotSet, ""), none, true},
		{"attribute equals, case insensitive", attr("tier", models.RuleOperatorEquals, "top"), top, true},
		{"attribute number", attr("seats", models.RuleOperatorGreaterThan, "10"), top, true},
		{"attribute bool", attr("vip", models.RuleOperatorEquals, "true"), top, true},
		{"attribute missing", attr("plan", models.RuleOperatorEquals, "x"), top, false},
		{"attribute not set", attr("plan", models.RuleOperatorNotSet, ""), top, true},
		{"no org attribute", attr("tier", models.RuleOperatorEquals, "top"), none, false},
		{"no org attribute not set", attr("tier", models.RuleOperatorNotSet, ""), none, true},
	}
	for _, tc := range cases {
		if got := e.evaluateRule(tc.rule, tc.conv, nil); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}

	// The contact's attributes are not the organization's.
	contactOnly := createTestConversation(func(c *cmodels.Conversation) {
		c.Contact.CustomAttributes = json.RawMessage(`{"tier":"top"}`)
	})
	if e.evaluateRule(attr("tier", models.RuleOperatorEquals, "top"), contactOnly, nil) {
		t.Error("organization attribute rule read the contact's attributes")
	}
}
