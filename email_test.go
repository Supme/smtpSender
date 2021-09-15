package smtpSender

import "testing"

type emailField struct {
	input, name, email, domain string
}

var (
	rightEmail []emailField
	badEmail   []emailField
)

func init() {
	rightEmail = append(rightEmail, emailField{" My name   <  my+email@domain.tld.  > ", "My name", "my+email", "domain.tld"})
	rightEmail = append(rightEmail, emailField{"  < My+Email@doMain.tld.  >  ", "", "my+email", "domain.tld"})
	rightEmail = append(rightEmail, emailField{"  mY+eMail@Domain.Tld.   ", "", "my+email", "domain.tld"})
	rightEmail = append(rightEmail, emailField{"recipient@linklocal.supme.ru", "", "recipient", "linklocal.supme.ru"})
	rightEmail = append(rightEmail, emailField{"=?utf-8?q?=D0=9E=D1=82=D0=BF=D1=80=D0=B0=D0=B2=D0=B8=D1=82=D0=B5=D0=BB?= =?utf-8?q?=D1=8C?= <sender@localhost.localdomain>", "=?utf-8?q?=D0=9E=D1=82=D0=BF=D1=80=D0=B0=D0=B2=D0=B8=D1=82=D0=B5=D0=BB?= =?utf-8?q?=D1=8C?=", "sender", "localhost.localdomain"})

	badEmail = append(badEmail, emailField{input: "my+email@domain.t"})
	badEmail = append(badEmail, emailField{input: "< my+email[at]domain.tld>"})
	//badEmail = append(badEmail, emailField{input: "<my+email@domain.tld."})
}

func TestSplitEmail(t *testing.T) {
	for _, v := range rightEmail {
		name, email, domain, err := splitEmail(v.input)
		if err != nil {
			t.Errorf("Email '%s' has error: %s", v.input, err)
		}
		if v.name != name {
			t.Errorf("Email '%s' not valid name: want '%s', has '%s'", v.input, v.name, name)
		}
		if v.email != email {
			t.Errorf("Email '%s' not valid email: want '%s', has '%s'", v.input, v.email, email)
		}
		if v.domain != domain {
			t.Errorf("Email '%s' not valid name: want '%s', has '%s'", v.input, v.domain, domain)
		}
	}

	for _, v := range badEmail {
		_, _, _, err := splitEmail(v.input)
		if err == nil {
			t.Errorf("Email '%s' has bad format, but parsed without error", v.input)
		}
	}
}

func BenchmarkSplitEmailFullString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if _, _, _, err := splitEmail(rightEmail[0].input); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkSplitEmailOnlyString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if _, _, _, err := splitEmail(rightEmail[0].input); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkSplitEmail(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if _, _, _, err := splitEmail(rightEmail[0].input); err != nil {
			b.Error(err)
		}
	}
}
