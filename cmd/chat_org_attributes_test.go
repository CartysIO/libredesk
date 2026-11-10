package main

import (
	"testing"

	customAttribute "github.com/abhinavxd/libredesk/internal/custom_attribute"
	cmodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

// Organization attributes in a pre-chat form are never shown to a visitor or saved from one (Clyr fork).
func TestPreChatFormIgnoresOrganizationAttributes(t *testing.T) {
	db := testutil.NewDB(t, "cmd_prechat_org_attributes")
	lo := logf.New(logf.Opts{Level: logf.FatalLevel})
	lang := testutil.NewI18n(t)
	mgr, err := customAttribute.New(customAttribute.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{lo: &lo, i18n: lang, customAttribute: mgr}
	ids := map[string]int{}
	for _, target := range []string{"contact", "conversation", "organization"} {
		attr, err := mgr.Create(cmodels.CustomAttribute{AppliesTo: target, Name: target, Description: target, Key: target + "_key", DataType: "text", Values: []string{}})
		if err != nil {
			t.Fatal(err)
		}
		ids[target] = attr.ID
	}

	var config livechat.Config
	config.PreChatForm.Enabled = true
	for target, id := range ids {
		config.PreChatForm.Fields = append(config.PreChatForm.Fields, livechat.PreChatFormField{Key: target + "_key", Type: "text", Enabled: true, CustomAttributeID: id})
	}

	contact, conversation := validateCustomAttributes(map[string]any{"contact_key": "a", "conversation_key": "b", "organization_key": "c"}, config, app)
	if len(contact) != 1 || contact["contact_key"] != "a" || len(conversation) != 1 || conversation["conversation_key"] != "b" {
		t.Fatalf("contact = %v, conversation = %v", contact, conversation)
	}

	fields, _ := filterPreChatFormFields(config.PreChatForm.Fields, app)
	for _, f := range fields {
		if f.CustomAttributeID == ids["organization"] {
			t.Fatal("organization attribute shown in the pre-chat form")
		}
	}
	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(fields))
	}
}
