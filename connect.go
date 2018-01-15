package smtpSender

import (
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

const connTimeout = 5 * time.Second // SMTP RFC

// ToDo cache connect to server ???
func (e *Email) connect() (conn *smtp.Client, err error) {
	dialFunc, err := e.dialFunction()
	if err != nil {
		return nil, err
	}

	conn, err = e.newClient(e.toDomain, dialFunc)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func (e *Email) newClient(server string, dialFunc func(network, address string) (net.Conn, error)) (client *smtp.Client, err error) {
	var conn net.Conn

	records, err := net.LookupMX(server)
	if err != nil {
		return
	}

	for i := range records {
		server := strings.TrimRight(strings.TrimSpace(records[i].Host), ".")
		conn, err = dialFunc("tcp", net.JoinHostPort(server, fmt.Sprintf("%d", e.portSMTP)))
		if err != nil {
			continue
		}
		err = conn.SetDeadline(time.Now().Add(connTimeout))
		if err != nil {
			return
		}
		err = conn.SetReadDeadline(time.Now().Add(connTimeout))
		if err != nil {
			return
		}
		err = conn.SetWriteDeadline(time.Now().Add(connTimeout))
		if err != nil {
			return
		}

		if e.ip == "" {
			e.ip = conn.LocalAddr().String()
		}

		if e.hostname == "" {
			var myGlobalIP string
			myIP, _, err := net.SplitHostPort(strings.TrimLeft(e.ip, "socks://"))
			myGlobalIP, ok := e.MapIP[myIP]
			if !ok {
				myGlobalIP = myIP
			}
			names, err := net.LookupAddr(myGlobalIP)
			if err != nil && len(names) < 1 {
				return nil, err
			}
			e.hostname = names[0]
		}

		client, err = smtp.NewClient(conn, server)
		if err != nil {
			continue
		}
		err = client.Hello(strings.TrimRight(e.hostname, "."))
		if err == nil {
			break
		}
	}
	if err != nil {
		return
	}

	return
}

//type conn func(network, address string) (net.Conn, error)
func (e *Email) dialFunction() (func(network, address string) (net.Conn, error), error) {
	var dialFunc func(network, address string) (net.Conn, error)
	if e.ip == "" {
		iface := net.Dialer{}
		dialFunc = iface.Dial
	} else {
		if strings.ToLower(e.ip[0:8]) == "socks://" || strings.ToLower(e.ip[0:9]) == "socks5://" {
			u, err := url.Parse(e.ip)
			if err != nil {
				return nil, fmt.Errorf("Error parse socks: %s", err.Error())
			}
			var iface proxy.Dialer
			if u.User != nil {
				auth := proxy.Auth{}
				auth.User = u.User.Username()
				auth.Password, _ = u.User.Password()
				iface, err = proxy.SOCKS5("tcp", u.Host, &auth, proxy.FromEnvironment())
				if err != nil {
					return dialFunc, err
				}
			} else {
				iface, err = proxy.SOCKS5("tcp", u.Host, nil, proxy.FromEnvironment())
				if err != nil {
					return dialFunc, err
				}
			}
			e.ip = u.Host
			dialFunc = iface.Dial
		} else {
			iface := net.Dialer{
				LocalAddr: &net.TCPAddr{
					IP: net.ParseIP(e.ip),
				},
			}
			dialFunc = iface.Dial
		}
	}

	return dialFunc, nil
}
