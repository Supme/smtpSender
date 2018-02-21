package smtpSender

import (
	"errors"
	"sync"
)

// Config profile for sender pool
type Config struct {
	Hostname   string
	Iface      string
	Port       int
	Stream     int
	MapIP      map[string]string
	SMTPserver SMTPserver
}

// Pipe email pipe for send email
type Pipe struct {
	email  chan<- Email
	config []Config
	finish chan struct{}
	stoped bool
	sync.Mutex
}

type sender struct {
	conf   Config
	backet chan struct{}
}

// NewPipe return new stream sender pipe
func NewPipe(conf ...Config) *Pipe {
	pipe := &Pipe{stoped: true}
	for _, c := range conf {
		pipe.config = append(pipe.config, c)
	}
	return pipe
}

// Start stream sender
func (pipe *Pipe) Start() {
	for _, c := range pipe.config {
		s := new(sender)
		s.backet = make(chan struct{}, c.Stream)
		s.conf = c
		go func(s *sender) {
			for e := range pipe.email {
				s.backet <- struct{}{}
				func(e *Email) {
					conn := new(Connect)
					conn.SetHostName(s.conf.Hostname)
					conn.SetSMTPport(s.conf.Port)
					conn.SetIface(s.conf.Iface)
					conn.mapIP = s.conf.MapIP
					e.Send(conn, &s.conf.SMTPserver)
					_ = <-s.backet
				}(&e)
			}
			close(s.backet)
			pipe.finish <- struct{}{}
		}(s)
	}
	pipe.Lock()
	pipe.stoped = false
	pipe.Unlock()
}

// Send add email to stream
func (pipe *Pipe) Send(email Email) error {
	pipe.Lock()
	defer pipe.Unlock()
	if !pipe.stoped {
		return errors.New("email streaming pipe stopped")
	}
	pipe.email <- email
	return nil
}

// Stop stream sender
func (pipe *Pipe) Stop() {
	pipe.Lock()
	pipe.stoped = true
	close(pipe.email)
	pipe.Unlock()
	for i := 0; i < len(pipe.config); i++ {
		_ = <-pipe.finish
	}
}

// NewEmailPipe return new chanel for stream send
func NewEmailPipe(conf ...Config) chan<- Email {
	pipe := NewPipe(conf...)
	pipe.Start()
	return pipe.email
}
