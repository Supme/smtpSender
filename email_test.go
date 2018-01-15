package smtpSender

import "testing"

func TestAverage(t *testing.T) {
	type emailField struct {
		input, name, email, domain string
	}

	rightEmail := []emailField{}
	rightEmail = append(rightEmail, emailField{" My name   <  my+email@domain.tld  > ", "My name", "my+email", "domain.tld"})
	rightEmail = append(rightEmail, emailField{"  < My+Email@doMain.tld  >  ", "", "my+email", "domain.tld"})
	rightEmail = append(rightEmail, emailField{"  mY+eMail@Domain.Tld   ", "", "my+email", "domain.tld"})

	for _, v := range rightEmail {
		name, email, domain := splitEmail(v.input)
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
}
