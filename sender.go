package smtpSender

import (
	"time"
)

// Result struct for return send emailField result
type Result struct {
	ID       string
	Err      error
	Duration time.Duration
}

type sender struct {
	conf   Config
	backet chan struct{}
}

// Config profile for sender pool
type Config struct {
	Hostname string
	IP       string
	Port     int
	Stream   int
	MapIP map[string]string
}

// NewEmailChan return new stream sender
func NewEmailChan(conf ...Config) chan Email {
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
					e.SetHostName(s.conf.Hostname)
					e.SetSMTPport(s.conf.Port)
					e.SetIP(s.conf.IP)
					e.MapIP = s.conf.MapIP
					e.Send()
					_ = <-s.backet
				}(&e)
			}
			close(s.backet)
		}(s)
	}
	return EmailChan
}
