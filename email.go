package smtpSender

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"regexp"
	"strings"
	"time"
)

// Email struct
type Email struct {
	// ID is id for return result
	ID string
	// From emailField has format
	// example
	//  "Name <emailField@domain.tld>"
	//  "<emailField@domain.tld>"
	//  "emailField@domain.tld"
	From                            string
	fromName, fromEmail, fromDomain string
	// To emailField has format as From
	To                        string
	toName, toEmail, toDomain string
	// ResultFunc exec after send emil
	ResultFunc func(Result)
	// Data email body data
	Data io.Reader

	ip       string
	hostname string
	portSMTP int
	// MapIP use for translate local IP to global if NAT
	// if use Socks server translate IP SOCKS server to real IP
	MapIP map[string]string
}

// SetHostName set server hostname for HELO. If left blanc then use resolv name.
func (e *Email) SetHostName(name string) {
	e.hostname = name
}

// SetSMTPport set SMTP server port. Default 25
func (e *Email) SetSMTPport(port int) {
	e.portSMTP = port
}

// SetIP use this IP for send. Default use default interface.
func (e *Email) SetIP(ip string) {
	e.ip = ip
}

// Send sending this email
func (e *Email) Send() {
	start := time.Now()
	if e.portSMTP == 0 {
		e.portSMTP = 25
	}
	e.parseEmail()
	conn, err := e.connect()
	if err != nil {
		e.ResultFunc(Result{e.ID, fmt.Errorf("421 %v", err), time.Now().Sub(start)})
		return
	}
	//defer conn.Close()
	err = e.send(nil, "", conn)
	e.ResultFunc(Result{e.ID, err, time.Now().Sub(start)})
	return
}

var testHookStartTLS func(*tls.Config)

func (e *Email) send(auth smtp.Auth, host string, client *smtp.Client) error {
	var err error

	if auth != nil {
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: host}
			if testHookStartTLS != nil {
				testHookStartTLS(config)
			}
			if err = client.StartTLS(config); err != nil {
				return err
			}
		}
		if auth != nil {
			if err = client.Auth(auth); err != nil {
				return err
			}
		}
	}

	if err := client.Mail(e.from()); err != nil {
		return err
	}

	if err := client.Rcpt(e.to()); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = io.Copy(w, e.Data)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	//client.Quit()
	return err

}

func (e *Email) from() string {
	return e.fromEmail + "@" + e.fromDomain
}

func (e *Email) to() string {
	return e.toEmail + "@" + e.toDomain
}

func (e *Email) parseEmail() {
	e.fromName, e.fromEmail, e.fromDomain = splitEmail(e.From)
	e.toName, e.toEmail, e.toDomain = splitEmail(e.To)
}

var (
	splitEmailFullStringRe = regexp.MustCompile(`(.+)<(.+)@(.+\..{2,8})>`)
	splitEmailOnlyStringRe = regexp.MustCompile(`<(.+)@(.+\..{2,8})>`)
	splitEmailRe           = regexp.MustCompile(`(.+)@(.+\..{2,8})`)
)

func splitEmail(e string) (name, email, domain string) {
	s := strings.TrimSpace(e)
	if m := splitEmailFullStringRe.FindStringSubmatch(s); m != nil && len(m) == 4 {
		name = strings.TrimSpace(m[1])
		email = strings.ToLower(strings.TrimSpace(m[2]))
		domain = strings.ToLower(strings.TrimSpace(m[3]))
	} else if m := splitEmailOnlyStringRe.FindStringSubmatch(s); m != nil && len(m) == 3 {
		email = strings.ToLower(strings.TrimSpace(m[1]))
		domain = strings.ToLower(strings.TrimSpace(m[2]))
	} else if m := splitEmailRe.FindStringSubmatch(s); m != nil && len(m) == 3 {
		email = strings.ToLower(strings.TrimSpace(m[1]))
		domain = strings.ToLower(strings.TrimSpace(m[2]))
	}

	return
}
