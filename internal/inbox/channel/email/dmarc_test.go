package email

import "testing"

func TestDMARCPass(t *testing.T) {
	const gmail = `mx.google.com; dkim=pass header.i=@acme.com header.s=s1 header.b=abc; spf=pass (google.com: domain of a@acme.com designates 1.2.3.4 as permitted sender) smtp.mailfrom=a@acme.com; dmarc=pass (p=REJECT sp=REJECT dis=NONE) header.from=acme.com`
	cases := []struct {
		name, header, domain string
		want                 bool
	}{
		{"gmail pass", gmail, "acme.com", true},
		{"case and spaces", "MX.EXAMPLE.NET;\r\n DMARC=PASS header.from=Acme.COM", "acme.com", true},
		{"other domain", gmail, "globex.com", false},
		{"dmarc fail", "mx.google.com; dmarc=fail (p=NONE) header.from=acme.com", "acme.com", false},
		{"dmarc none", "mx.google.com; dmarc=none header.from=acme.com", "acme.com", false},
		{"no dmarc", "mx.google.com; spf=pass smtp.mailfrom=acme.com", "acme.com", false},
		{"pass only inside a comment", "mx.google.com; spf=pass (dmarc=pass header.from=acme.com) smtp.mailfrom=x.com", "acme.com", false},
		{"subdomain is not the domain", "mx.google.com; dmarc=pass header.from=mail.acme.com", "acme.com", false},
		{"authserv-id only", "dmarc=pass header.from=acme.com", "acme.com", false},
		{"empty", "", "acme.com", false},
		{"no from domain", gmail, "", false},
	}
	for _, tc := range cases {
		if got := dmarcPass(tc.header, tc.domain); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
