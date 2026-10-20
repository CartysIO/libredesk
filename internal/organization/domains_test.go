package organization

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/organization/models"
	"github.com/volatiletech/null/v9"
)

func TestNormalizeDomains(t *testing.T) {
	got, bad := normalizeDomains([]string{" Acme.COM ", "@acme.com", "", "sub.acme.co.uk.", "xn--bcher-kva.example"})
	if bad != "" || !reflect.DeepEqual(got, []string{"acme.com", "sub.acme.co.uk", "xn--bcher-kva.example"}) {
		t.Fatalf("got %v, bad %q", got, bad)
	}
	if got, _ := normalizeDomains(nil); got != nil {
		t.Fatal("nil input must stay nil so stored domains are kept")
	}
	if got, _ := normalizeDomains([]string{}); got == nil || len(got) != 0 {
		t.Fatal("empty input must be an empty list so domains can be cleared")
	}
	for _, d := range []string{"acme", "a b.com", "-acme.com", "acme..com", "acme.com/x", "ann@acme.com"} {
		if _, bad := normalizeDomains([]string{d}); bad == "" {
			t.Errorf("%q accepted", d)
		}
	}
}

func TestEmailDomain(t *testing.T) {
	for in, want := range map[string]string{"Ann@Acme.COM": "acme.com", "x@y@acme.com": "acme.com", "no-at": "", "a@": ""} {
		if got := EmailDomain(in); got != want {
			t.Errorf("EmailDomain(%q) = %q, want %q", in, got, want)
		}
	}
	if !IsFreeMailDomain("GMAIL.com") || IsFreeMailDomain("acme.com") {
		t.Fatal("IsFreeMailDomain")
	}
}

func TestSetFields(t *testing.T) {
	mgr, _ := newTestManager(t)
	org, _ := mgr.Create("Acme", "team_1")
	if string(org.CustomAttributes) != "{}" || len(org.Domains) != 0 {
		t.Fatalf("new organization fields = %s %v", org.CustomAttributes, org.Domains)
	}

	got, err := mgr.SetFields(org.ID, models.Fields{Domains: []string{"Acme.com"}, CustomAttributes: map[string]any{"tier": "top", "seats": 10}}, false)
	if err != nil {
		t.Fatalf("SetFields: %v", err)
	}
	assertFields(t, got, []string{"acme.com"}, map[string]any{"tier": "top", "seats": float64(10)})

	// A merge adds domains and keys, and keeps the others.
	got, err = mgr.SetFields(org.ID, models.Fields{Domains: []string{"acme.io", "acme.com"}, CustomAttributes: map[string]any{"seats": 12}}, true)
	if err != nil {
		t.Fatalf("SetFields merge: %v", err)
	}
	assertFields(t, got, []string{"acme.com", "acme.io"}, map[string]any{"tier": "top", "seats": float64(12)})

	// Nil fields are kept; given fields replace.
	got, _ = mgr.SetFields(org.ID, models.Fields{CustomAttributes: map[string]any{"plan": "pro"}}, false)
	assertFields(t, got, []string{"acme.com", "acme.io"}, map[string]any{"plan": "pro"})
	got, _ = mgr.SetFields(org.ID, models.Fields{Domains: []string{}}, false)
	assertFields(t, got, []string{}, map[string]any{"plan": "pro"})

	if _, err := mgr.SetFields(org.ID, models.Fields{Domains: []string{"not a domain"}}, false); errType(err) != envelope.InputError {
		t.Fatalf("invalid domain: %v", err)
	}
	if _, err := mgr.SetFields(999999, models.Fields{Domains: []string{"x.com"}}, false); errType(err) != envelope.NotFoundError {
		t.Fatalf("missing organization: %v", err)
	}
}

func assertFields(t *testing.T, org models.Organization, domains []string, attrs map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(org.CustomAttributes, &got); err != nil {
		t.Fatalf("custom_attributes %s: %v", org.CustomAttributes, err)
	}
	if !reflect.DeepEqual([]string(org.Domains), domains) || !reflect.DeepEqual(got, attrs) {
		t.Fatalf("got %v %v, want %v %v", org.Domains, got, domains, attrs)
	}
}

func TestAssignByEmailDomain(t *testing.T) {
	mgr, db := newTestManager(t)
	acme, _ := mgr.Create("Acme", "team_1")
	globex, _ := mgr.Create("Globex", "team_2")
	other, _ := mgr.Create("Other", "team_3")
	mgr.SetFields(acme.ID, models.Fields{Domains: []string{"acme.com", "gmail.com"}}, false)
	mgr.SetFields(globex.ID, models.Fields{Domains: []string{"globex.com", "shared.com"}}, false)
	mgr.SetFields(other.ID, models.Fields{Domains: []string{"shared.com"}}, false)

	ann := seedContact(t, db, "ann")
	conv := seedConversation(t, db, ann, null.Int{})
	orgID, filed, err := mgr.AssignByEmailDomain(ann, "Ann@ACME.com")
	if err != nil || orgID != acme.ID || len(filed) != 1 {
		t.Fatalf("AssignByEmailDomain = %d %v %v", orgID, filed, err)
	}
	orgs, _ := mgr.GetContactOrganizations(ann)
	if len(orgs) != 1 || orgs[0].ID != acme.ID || !orgs[0].IsDefault {
		t.Fatalf("ann organizations = %+v", orgs)
	}
	var convOrg null.Int
	db.Get(&convOrg, `SELECT organization_id FROM conversations WHERE id = $1`, conv)
	if convOrg.Int != acme.ID {
		t.Fatalf("conversation organization = %v", convOrg)
	}

	// A contact that already has an organization keeps it, even for another domain.
	if orgID, _, _ := mgr.AssignByEmailDomain(ann, "ann@globex.com"); orgID != 0 {
		t.Fatal("existing membership was overridden")
	}
	bob := seedContact(t, db, "bob")
	mgr.AddContactOrganization(bob, other.ID, false)
	if orgID, _, _ := mgr.AssignByEmailDomain(bob, "bob@globex.com"); orgID != 0 {
		t.Fatal("contact with a membership was assigned")
	}

	// Free mail, a domain two organizations share, and an unknown domain do nothing.
	cara := seedContact(t, db, "cara")
	for _, email := range []string{"cara@gmail.com", "cara@shared.com", "cara@unknown.com", "no-domain"} {
		if orgID, _, err := mgr.AssignByEmailDomain(cara, email); orgID != 0 || err != nil {
			t.Fatalf("%s: assigned %d, %v", email, orgID, err)
		}
	}
	if orgs, _ := mgr.GetContactOrganizations(cara); len(orgs) != 0 {
		t.Fatalf("cara organizations = %+v", orgs)
	}
}
