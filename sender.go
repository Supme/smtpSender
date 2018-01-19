package smtpSender

import (
	"time"
)

// Result struct for return send emailField result
type Result struct {
	ID       string
	Duration time.Duration
	Err      error
}

type sender struct {
	conf   Config
	backet chan struct{}
}

// Config profile for sender pool
type Config struct {
	Hostname string
	Iface    string
	Port     int
	Stream   int
	MapIP    map[string]string
}

// NewEmailPipe return new stream sender
func NewEmailPipe(conf ...Config) chan Email {
	EmailChan := make(chan Email)
	for _, c := range conf {
		s := new(sender)
		s.backet = make(chan struct{}, c.Stream)
		s.conf = c
		go func(s *sender) {
			for e := range EmailChan {
				s.backet <- struct{}{}
				func(e *Email) {
					//log.Printf("Send email id: %s from profile with %d stream\n", e.ID, s.conf.Stream)
					conn := new(Connect)
					conn.SetHostName(s.conf.Hostname)
					conn.SetSMTPport(s.conf.Port)
					conn.SetIface(s.conf.Iface)
					conn.mapIP = s.conf.MapIP
					e.Send(conn)
					_ = <-s.backet
				}(&e)
			}
			close(s.backet)
		}(s)
	}
	return EmailChan
}
