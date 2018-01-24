package smtpSender

import (
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"mime/quotedprintable"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Builder helper for create email
type Builder struct {
	From            string
	To              string
	Subject         string
	replyTo         string
	headers         []string
	textPlain       []byte
	textHTML        []byte
	textHTMLRelated []*os.File
	attachments     []*os.File
	markerGlobal    marker
	markerAlt       marker
	markerHTML      marker
}

// SetFrom email sender
func (c *Builder) SetFrom(name, email string) {
	c.From = mime.BEncoding.Encode("utf-8", name) + "<" + email + ">"
}

// SetTo email recipient
func (c *Builder) SetTo(name, email string) {
	c.To = mime.BEncoding.Encode("utf-8", name) + "<" + email + ">"
}

// SetSubject set email subject
func (c *Builder) SetSubject(text string) {
	c.Subject = mime.BEncoding.Encode("utf-8", text)
}

// AddReplyTo add Reply-To header
func (c *Builder) AddReplyTo(name, email string) {
	c.replyTo = email
}

// AddHeader add extra header to email
func (c *Builder) AddHeader(headers ...string) {
	for i := range headers {
		c.headers = append(c.headers, headers[i]+"\r\n")
	}
}

// AddTextHTML add text/html content with related file.
//
// Example use related file in html
//  AddTextHTML(
//  	`... <img src="cid:myImage.jpg" width="500px" height="250px" border="1px" alt="My image"/> ...`,
//  	"/path/to/attach/myImage.jpg",
//  )
func (c *Builder) AddTextHTML(html []byte, files ...string) (err error) {
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

// AddTextPlain add plain text
func (c *Builder) AddTextPlain(text []byte) {
	c.textPlain = text
}

// AddAttachment add attachment files to email
func (c *Builder) AddAttachment(files ...string) error {
	for i := range files {
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		c.attachments = append(c.attachments, file)
	}
	return nil
}

// Email return Email struct with render function
func (c *Builder) Email(id string, resultFunc func(Result)) Email {
	email := new(Email)
	email.ID = id
	email.From = c.From
	email.To = c.To
	email.ResultFunc = resultFunc
	email.Writer = func(w io.Writer) (err error) {
		// Headers
		err = c.writeHeaders(w)
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

		if len(c.attachments) != 0 {
			c.markerGlobal.new()
			_, err = w.Write([]byte("Content-Type: multipart/mixed;\r\n\tboundary=\"" + c.markerGlobal.string() + "\"\r\n"))
			if err != nil {
				return
			}
		}

		// Plain text this Text HTML
		if len(c.textPlain) != 0 && len(c.textHTML) != 0 {
			if c.markerGlobal.isset() {
				_, err = w.Write([]byte("\r\n"))
				if err != nil {
					return
				}
				_, err = w.Write(c.markerGlobal.delimiter())
				if err != nil {
					return
				}
			}
			c.markerAlt.new()
			_, err = w.Write([]byte("Content-Type: multipart/alternative;\r\n\tboundary=\"" + c.markerAlt.string() + "\"\r\n\r\n"))
			if err != nil {
				return
			}

			_, err = w.Write(c.markerAlt.delimiter())
			if err != nil {
				return
			}
			err = c.writeTextPlain(w)
			if err != nil {
				return
			}

			_, err = w.Write(c.markerAlt.delimiter())
			if err != nil {
				return
			}
			err = c.writeTextHTML(w)
			if err != nil {
				return
			}

			_, err = w.Write(c.markerAlt.finish())
			if err != nil {
				return
			}

		} else if len(c.textHTML) != 0 {
			if c.markerGlobal.isset() {
				_, err = w.Write([]byte("\r\n"))
				if err != nil {
					return
				}
				_, err = w.Write(c.markerGlobal.delimiter())
				if err != nil {
					return
				}
			}
			err = c.writeTextHTML(w)
			if err != nil {
				return
			}
		} else if len(c.textPlain) != 0 {
			if c.markerGlobal.isset() {
				_, err = w.Write([]byte("\r\n"))
				if err != nil {
					return
				}
				_, err = w.Write(c.markerGlobal.delimiter())
				if err != nil {
					return
				}
			}
			err = c.writeTextPlain(w)
			if err != nil {
				return
			}
		}

		// Attachments
		err = c.writeAttachment(w)
		if err != nil {
			return
		}

		if c.markerGlobal.isset() {
			_, err = w.Write(c.markerGlobal.finish())
			if err != nil {
				return
			}
		}
		return
	}
	return *email
}

func (c *Builder) writeHeaders(w io.Writer) (err error) {
	_, err = w.Write([]byte("From: " + c.From + "\r\n"))
	if err != nil {
		return
	}
	_, err = w.Write([]byte("To: " + c.To + "\r\n"))
	if err != nil {
		return
	}
	if c.replyTo != "" {
		_, err = w.Write([]byte("Reply-To: <" + c.replyTo + ">\r\n"))
		if err != nil {
			return
		}
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
	_, err = w.Write([]byte("Subject: " + c.Subject + "\r\n"))
	return
}

// Text block
func (c *Builder) writeTextPlain(w io.Writer) (err error) {
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
	if len(c.textHTMLRelated) != 0 {
		c.markerHTML.new()
		_, err = w.Write([]byte("Content-Type: multipart/related;\r\n\tboundary=\"" + c.markerHTML.string() + "\"\r\n\r\n"))
		if err != nil {
			return
		}
		_, err = w.Write(c.markerHTML.delimiter())
		if err != nil {
			return
		}
	}
	_, err = w.Write([]byte("Content-Type: text/html;\r\n\t charset=\"utf-8\"\r\nContent-Transfer-Encoding: base64\r\n\r\n"))
	if err != nil {
		return
	}
	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 76) // 76 from RFC
	b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)
	_, err = b64Enc.Write(c.textHTML)
	if err != nil {
		return
	}
	err = b64Enc.Close()
	if err != nil {
		return
	}
	_, err = w.Write([]byte("\r\n"))
	if err != nil {
		return
	}

	// related files
	for i := range c.textHTMLRelated {
		_, err = w.Write([]byte("\r\n"))
		if err != nil {
			return
		}
		_, err = w.Write(c.markerHTML.delimiter())
		if err != nil {
			return
		}

		err = fileBase64Writer(w, c.textHTMLRelated[i], "inline")
		if err != nil {
			return
		}
		_, err = w.Write([]byte("\r\n\r\n"))
		if err != nil {
			return
		}
	}

	if c.markerHTML.isset() {
		_, err = w.Write(c.markerHTML.finish())
		if err != nil {
			return
		}
	}
	return
}

func (c *Builder) writeAttachment(w io.Writer) (err error) {
	for i := range c.attachments {
		if !c.markerGlobal.isset() {
			c.markerGlobal.new()
			_, err = w.Write([]byte("Content-Type: multipart/mixed;\r\n\tboundary=\"" + c.markerGlobal.string() + "\"\r\n"))
			if err != nil {
				return
			}
		}
		_, err = w.Write([]byte("\r\n"))
		if err != nil {
			return
		}
		_, err = w.Write(c.markerGlobal.delimiter())
		if err != nil {
			return
		}
		err = fileBase64Writer(w, c.attachments[i], "attachment")
		if err != nil {
			return
		}
		_, err = w.Write([]byte("\r\n\r\n"))
		if err != nil {
			return
		}
	}
	return
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
		if w.n++; w.n%w.cnt == 0 {
			w.writer.Write(w.dr)

		}
	}
	return w.n, err
}

func fileBase64Writer(w io.Writer, f *os.File, disposition string) (err error) {
	name := filepath.Base(f.Name())
	var info os.FileInfo
	info, err = f.Stat()
	if err != nil {
		return
	}
	size := info.Size()
	buf := make([]byte, 512)
	_, err = f.Read(buf)
	if err != nil && err != io.EOF {
		return
	}
	content := http.DetectContentType(buf)
	_, err = f.Seek(0, 0)
	var contentID string
	if disposition == "inline" {
		contentID = "Content-ID: <" + name + ">\r\n"
	}
	_, err = w.Write([]byte(fmt.Sprintf(
		"Content-Type: %s;\r\n\tname=\"%s\"\r\nContent-Transfer-Encoding: base64\r\n%sContent-Disposition: %s;\r\n\tfilename=\"%s\"; size=%d;\r\n\r\n",
		content,
		name,
		contentID,
		disposition,
		name,
		size)))
	if err != nil {
		return
	}

	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 76) // 76 from RFC
	b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)

	_, err = io.Copy(b64Enc, f)
	if err != nil {
		return err
	}

	return b64Enc.Close()
}

type marker []byte

func (m *marker) new() {
	b := make([]byte, 30)
	rand.Read(b)
	en := base64.StdEncoding // or URLEncoding
	d := make([]byte, en.EncodedLen(len(b)))
	en.Encode(d, b)
	*m = []byte(string(d))
}

func (m *marker) delimiter() []byte {
	return []byte("--" + string(*m) + "\r\n")
}

func (m *marker) finish() []byte {
	return []byte("\r\n--" + string(*m) + "--\r\n")
}

func (m *marker) isset() bool {
	return string(*m) != ""
}

func (m *marker) string() string {
	return string(*m)
}
