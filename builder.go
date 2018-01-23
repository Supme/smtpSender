package smtpSender

import (
	"encoding/base64"
	"io"
	"math/rand"
	"mime"
	"mime/quotedprintable"
	"os"
	"time"
)

type Builder struct {
	from            string
	to              string
	subject         string
	headers         []string
	textPlain       []byte
	textHTML        []byte
	textHTMLRelated []*os.File
	attachments     []*os.File
	markerGlobal    marker
	markerHTML      marker
}

type marker []byte

func (m *marker) new() {
	b := make([]byte, 30)
	rand.Read(b)
	en := base64.StdEncoding // or URLEncoding
	d := make([]byte, en.EncodedLen(len(b)))
	en.Encode(d, b)
	*m = []byte("--_" + string(d) + "_")
}

func (m *marker) delemiter() []byte {
	return []byte(string(*m) +"\r\n")
}

func (m *marker) finish() []byte {
	return []byte("\r\n" + string(*m) + "--\r\n")
}

func (m *marker) isset() bool {
	return string(*m) != ""
}

func (m *marker) string() string {
	return string(*m)
}


func (c *Builder) Render(id string, resultFunc func(Result)) *Email {
	var email Email
	email.ID = id
	email.From = c.from
	email.To = c.to
	email.ResultFunc = resultFunc
	email.Writer = func(w io.Writer) (err error) {
		// Headers
		err = c.WriteHeaders(w)
		if err != nil {
			return
		}
		blkCount := 0
		if len(c.attachments) != 0 {
			blkCount++
		}
		if len(c.textPlain) != 0 {
			blkCount++
		}
		if len(c.textHTML) != 0 {
			blkCount++
		}

		if blkCount > 1 {
			c.markerGlobal.new()
			_, err = w.Write([]byte("Content-Type: multipart/mixed;\r\n\tboundary=\"" + c.markerGlobal.string() + "\"\r\n"))
			if err != nil {
				return
			}
		}

		// Text HTML
		if len(c.textHTML) != 0 {
			err = c.writeTextHTML(w)
			if err != nil {
				return
			}
		}

		// Plain text
		if len(c.textPlain) != 0 {
			err = c.writeTextPlain(w)
			if err != nil {
				return
			}
		}


		if c.markerGlobal.isset() {
			_, err = w.Write(c.markerGlobal.finish())
			if err != nil {
				return
			}
		}
		return
	}
	return &email
}

// Text block
func (c *Builder) writeTextPlain(w io.Writer) (err error) {
	if c.markerGlobal.isset() {
		_, err = w.Write([]byte("\r\n"))
		if err != nil {
			return
		}
		_, err = w.Write(c.markerGlobal.delemiter())
		if err != nil {
			return
		}
	}
	_, err = w.Write([]byte("Content-Type: text/plain;\r\n\t charset=\"utf-8\"\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n"))
	if err != nil {
		return
	}
	q := quotedprintable.NewWriter(w)
	_, err = q.Write(c.textPlain)
	if err != nil {
		return
	}
	q.Close()
	_, err = w.Write([]byte("\r\n"))
	if err != nil {
		return
	}
	return q.Close()
}

// HTML block
func (c *Builder) writeTextHTML(w io.Writer) (err error) {
	if c.markerGlobal.isset() {
		_, err = w.Write([]byte("\r\n"))
		if err != nil {
			return
		}
		_, err = w.Write(c.markerGlobal.delemiter())
		if err != nil {
			return
		}
	}
	if len(c.textHTMLRelated) != 0 {
		c.markerHTML.new()
		_, err = w.Write([]byte("Content-Type: multipart/related;\r\n\tboundary=\"" + c.markerHTML.string() + "\"\r\n\\r\n"))
		if err != nil {
			return
		}
		_, err = w.Write(c.markerHTML.delemiter())
		if err != nil {
			return
		}
	}
	_, err = w.Write([]byte("Content-Type: text/html;\r\n\t charset=\"utf-8\"\r\nContent-Transfer-Encoding: base64\r\n\r\n"))
	if err != nil {
		return
	}
	dwr := newDelimitWriter(w, []byte{0x0d,0x0a}, 76) // 76 from RFC
	encoder := base64.NewEncoder(base64.StdEncoding, dwr)
	_, err = encoder.Write(c.textHTML)
	if err != nil {
		return
	}
	_, err = w.Write([]byte("\r\n"))
	if err != nil {
		return
	}
	if c.markerHTML.isset() {
		_, err = w.Write(c.markerHTML.finish())
		if err != nil {
			return
		}
	}
	return
}

func (c *Builder) WriteHeaders(w io.Writer) (err error) {
	_, err = w.Write([]byte("Return-path: " + c.from + "\r\n"))
	if err != nil {
		return
	}
	_, err = w.Write([]byte("From: " + c.from + "\r\n"))
	if err != nil {
		return
	}
	_, err = w.Write([]byte("To: " + c.to + "\r\n"))
	if err != nil {
		return
	}
	_, err = w.Write([]byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n"))
	if err != nil {
		return
	}
	_, err = w.Write([]byte("MIME-Version: 1.0\r\n"))
	for i := range c.headers {
		_, err = w.Write([]byte(c.headers[i]))
		if err != nil {
			return
		}
	}
	_, err = w.Write([]byte("Subject: " + c.subject + "\r\n"))
	return
}

func (c *Builder) From(name, email string) {
	c.from = mime.BEncoding.Encode("utf-8", name) + "<" + email + ">"
}

func (c *Builder) To(name, email string) {
	c.to = mime.BEncoding.Encode("utf-8", name) + "<" + email + ">"
}

func (c *Builder) Subject(text string) {
	c.subject = mime.BEncoding.Encode("utf-8", text)
}

func (c *Builder) Header(header ...string) {
	for i := range header {
		c.headers = append(c.headers, header[i]+"\r\n")
	}
}

// TextHtmlWithRelated add text/html content with related file.
//
// Example use file in html
//  email.TextHtmlWithRelated(
//  	`... <img src="cid:myImage.jpg" width="500px" height="250px" border="1px" alt="My image"/> ...`,
//  	"/path/to/attach/myImage.jpg",
//  )
func (c *Builder) TextHtmlWithRelated(html []byte, files ...string) (err error) {
	for i := range files {
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		c.textHTMLRelated = append(c.textHTMLRelated, file)
	}
	c.textHTML = html
	return nil
}
func (c *Builder) TextPlain(text []byte) {
	c.textPlain = text
}

func (c *Builder) Attachment(files ...string) error {
	for i := range files {
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		c.attachments = append(c.attachments, file)
	}
	return nil
}

type base64WriteCloser struct {
	encoder io.WriteCloser
}

func newBase64Writer(enc *base64.Encoding, w io.Writer) *base64WriteCloser {
	return &base64WriteCloser{encoder: base64.NewEncoder(enc, w)}
}

func (w *base64WriteCloser) Write(p []byte) (n int, err error) {
	return w.encoder.Write(p)
}

func (w *base64WriteCloser) Close() error {
	return w.encoder.Close()
}

type delimitWriter struct {
	n      int
	cnt    int
	dr     []byte
	writer io.Writer
}

func newDelimitWriter(writer io.Writer, dr []byte, cnt int) *delimitWriter {
	return &delimitWriter{n: 0, cnt: cnt, dr: dr, writer: writer}
}

func (w *delimitWriter) Write(p []byte) (n int, err error) {
	for i := range p {
		_, err = w.writer.Write(p[i : i+1])
		if err != nil {
			break
		}
		if w.n++; w.n % w.cnt == 0 {
			w.writer.Write(w.dr)

		}
	}
	return w.n, err
}
