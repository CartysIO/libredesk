package organization

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/organization/models"
	"github.com/lib/pq"
)

const (
	maxDomains                = 100
	maxCustomAttributesLength = 64 * 1024
)

var domainRe = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9-]{2,63}$`)

// freeMailDomains are email providers that many unrelated people use. A sender on one of them is never
// matched to an organization by domain.
// ponytail: fixed list of the big providers; move it to a setting if admins need to edit it.
var freeMailDomains = map[string]bool{
	"gmail.com": true, "googlemail.com": true,
	"outlook.com": true, "hotmail.com": true, "live.com": true, "msn.com": true,
	"yahoo.com": true, "ymail.com": true, "rocketmail.com": true,
	"icloud.com": true, "me.com": true, "mac.com": true,
	"aol.com": true, "proton.me": true, "protonmail.com": true, "pm.me": true,
	"gmx.com": true, "gmx.net": true, "mail.com": true, "zoho.com": true, "yandex.com": true,
	"comcast.net": true, "att.net": true, "sbcglobal.net": true, "verizon.net": true,
	"hotmail.co.uk": true, "yahoo.co.uk": true, "outlook.fr": true, "hotmail.fr": true,
}

// IsFreeMailDomain reports whether domain is a public email provider.
func IsFreeMailDomain(domain string) bool {
	return freeMailDomains[strings.ToLower(domain)]
}

// EmailDomain returns the lowercase domain of an email address, or "" when it has none.
func EmailDomain(email string) string {
	i := strings.LastIndex(email, "@")
	if i < 0 {
		return ""
	}
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(email[i+1:])), ".")
}

// normalizeDomains lowercases and trims domains, drops a leading "@" and empties, and removes duplicates.
// It returns the first invalid domain when there is one. A nil input stays nil, which keeps the stored domains.
func normalizeDomains(in []string) ([]string, string) {
	if in == nil {
		return nil, ""
	}
	out := make([]string, 0, len(in))
	for _, d := range in {
		d = strings.TrimSuffix(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(d)), "@"), ".")
		if d == "" {
			continue
		}
		if len(d) > 253 || !domainRe.MatchString(d) {
			return nil, d
		}
		if !slices.Contains(out, d) {
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return out, ""
}

// SetFields sets the domains and custom attributes of an organization; a nil field is left unchanged.
// With merge, domains are added and attributes merged into the current ones, as an external sync does;
// otherwise they replace the current ones.
func (m *Manager) SetFields(id int, f models.Fields, merge bool) (models.Organization, error) {
	var org models.Organization
	domains, attrs, err := m.ValidateFields(f)
	if err != nil {
		return org, err
	}
	var gotID int
	if err := m.q.SetOrganizationFields.Get(&gotID, id, domains, attrs, merge); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return org, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundOrganization"), nil)
		}
		m.lo.Error("error setting organization fields", "id", id, "error", err)
		return org, m.generalError()
	}
	return m.Get(id)
}

// ValidateFields checks the fields and returns them as query arguments: NULL for a field that is left unchanged.
func (m *Manager) ValidateFields(f models.Fields) (any, any, error) {
	domains, bad := normalizeDomains(f.Domains)
	if bad != "" {
		return nil, nil, envelope.NewError(envelope.InputError, m.i18n.Ts("organization.invalidDomain", "domain", bad), nil)
	}
	if len(domains) > maxDomains {
		return nil, nil, envelope.NewError(envelope.InputError, m.i18n.Ts("organization.tooManyDomains", "max", strconv.Itoa(maxDomains)), nil)
	}
	var domainsArg, attrsArg any
	if domains != nil {
		domainsArg = pq.StringArray(domains)
	}
	if f.CustomAttributes != nil {
		b, err := json.Marshal(f.CustomAttributes)
		if err != nil {
			return nil, nil, envelope.NewError(envelope.InputError, m.i18n.T("errors.parsingRequest"), nil)
		}
		if len(b) > maxCustomAttributesLength {
			return nil, nil, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.maxLength", "max", strconv.Itoa(maxCustomAttributesLength)), nil)
		}
		attrsArg = string(b)
	}
	return domainsArg, attrsArg, nil
}

// AssignByEmailDomain adds a contact that has no organization to the one organization that has the domain of
// email, and makes it the default. It never touches a contact that already has an organization, and ignores
// free-mail domains and domains that more than one organization has. It returns the organization ID (0 when
// nothing changed) and the conversations of the contact that were filed under it.
func (m *Manager) AssignByEmailDomain(contactID int, email string) (int, []string, error) {
	domain := EmailDomain(email)
	if domain == "" || IsFreeMailDomain(domain) {
		return 0, nil, nil
	}
	var res struct {
		OrganizationID int            `db:"organization_id"`
		Filed          pq.StringArray `db:"filed"`
	}
	if err := m.q.AssignOrganizationByDomain.Get(&res, contactID, domain); err != nil {
		m.lo.Error("error assigning organization by email domain", "contact_id", contactID, "domain", domain, "error", err)
		return 0, nil, m.generalError()
	}
	return res.OrganizationID, res.Filed, nil
}
