package main

import (
	"encoding/json"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/custom_attribute/models"
)

func TestStripPreChatFormAttribute(t *testing.T) {
	raw := json.RawMessage(`{
		"brand_name":"Acme",
		"prechat_form":{
			"fields":[{"key":"name"},{"key":"plan","custom_attribute_id":7}],
			"visitors":{"fields":[{"key":"plan","custom_attribute_id":7},{"key":"email"}]},
			"users":{"fields":[{"key":"email"}]}
		}
	}`)

	updated, changed, err := stripPreChatFormAttribute(raw, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected config to change")
	}

	var config struct {
		BrandName   string `json:"brand_name"`
		PreChatForm struct {
			Fields   []map[string]any `json:"fields"`
			Visitors struct {
				Fields []map[string]any `json:"fields"`
			} `json:"visitors"`
			Users struct {
				Fields []map[string]any `json:"fields"`
			} `json:"users"`
		} `json:"prechat_form"`
	}
	if err := json.Unmarshal(updated, &config); err != nil {
		t.Fatal(err)
	}
	if config.BrandName != "Acme" {
		t.Fatalf("brand name = %q", config.BrandName)
	}
	if len(config.PreChatForm.Fields) != 1 || config.PreChatForm.Fields[0]["key"] != "name" {
		t.Fatalf("fields = %v", config.PreChatForm.Fields)
	}
	if len(config.PreChatForm.Visitors.Fields) != 1 || config.PreChatForm.Visitors.Fields[0]["key"] != "email" {
		t.Fatalf("visitor fields = %v", config.PreChatForm.Visitors.Fields)
	}
	if len(config.PreChatForm.Users.Fields) != 1 {
		t.Fatalf("user fields = %v", config.PreChatForm.Users.Fields)
	}

	if _, changed, err := stripPreChatFormAttribute(raw, 8); err != nil || changed {
		t.Fatalf("unused attribute changed = %v, err = %v", changed, err)
	}
	if _, changed, err := stripPreChatFormAttribute(json.RawMessage(`{}`), 7); err != nil || changed {
		t.Fatalf("config without a pre-chat form changed = %v, err = %v", changed, err)
	}
}

func TestValidateCustomAttributeMultiSelect(t *testing.T) {
	app := newValidatorTestApp(t)
	base := cmodels.CustomAttribute{Name: "Question type", AppliesTo: "conversation", DataType: cmodels.DataTypeMultiSelect, Description: "What the customer asks", Key: "question_type"}

	ok := base
	ok.Values = []string{"Bug", "User Error"}
	if err := validateCustomAttribute(app, ok); err != nil {
		t.Fatalf("valid multi_select rejected: %v", err)
	}
	if err := validateCustomAttribute(app, base); err == nil {
		t.Fatal("multi_select without values must be rejected")
	}
	contact := ok
	contact.AppliesTo = "contact"
	if err := validateCustomAttribute(app, contact); err == nil {
		t.Fatal("multi_select on a contact attribute must be rejected")
	}
}
