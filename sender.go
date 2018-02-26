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
	SMTPserver *SMTPserver
}

// Pipe email pipe for send email
type Pipe struct {
	email  chan Email
	config []Config
	finish chan bool
}

// NewPipe return new stream sender pipe
func NewPipe(conf ...Config) Pipe {
	pipe := Pipe{}
	pipe.finish = make(chan bool, 1)
	for _, c := range conf {
		pipe.config = append(pipe.config, c)
	}
	pipe.email = make(chan Email, len(conf))
	return pipe
}

// Start stream sender
func (pipe *Pipe) Start() {
	go func() {
		wg := &sync.WaitGroup{}

		for i := range pipe.config {
			wg.Add(1)
			go func(conf *Config) {
				backet := make(chan struct{}, conf.Stream)
				for email := range pipe.email {
					backet <- struct{}{}
					go func(e Email) {
						conn := new(Connect)
						conn.SetHostName(conf.Hostname)
						conn.SetSMTPport(conf.Port)
						conn.SetIface(conf.Iface)
						conn.mapIP = conf.MapIP
						e.Send(conn, conf.SMTPserver)
						_ = <-backet
					}(email)
				}
				close(backet)
				wg.Done()
			}(&pipe.config[i])
		}

		wg.Wait()
		close(pipe.finish)
	}()
}

// Send add email to stream
func (pipe *Pipe) Send(email Email) {
	defer func(e *Email) {
		if err := recover(); err != nil {
			e.ResultFunc(Result{ID: e.ID, Err: errors.New("421 email streaming pipe stopped")})
		}
	}(&email)
	pipe.email <- email
}

// Stop stream sender
func (pipe *Pipe) Stop() {
	close(pipe.email)
	_ = <-pipe.finish
}

// NewEmailPipe return new chanel for stream send
func NewEmailPipe(conf ...Config) chan<- Email {
	pipe := NewPipe(conf...)
	pipe.Start()
	return pipe.email
}
