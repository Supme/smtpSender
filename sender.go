package smtpSender

type sender struct {
	conf   Config
	backet chan struct{}
}

// Config profile for sender pool
type Config struct {
	Hostname   string
	Iface      string
	Port       int
	Stream     int
	MapIP      map[string]string
	SMTPserver SMTPserver
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
		}(s)
	}
	return EmailChan
}
