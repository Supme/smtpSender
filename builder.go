package smtpSender

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/emersion/go-msgauth/dkim"
	"golang.org/x/crypto/ed25519"
	"io"
	"mime"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	boundaryMixed            = "===============_MIXED=="
	boundaryMixedBegin       = "--" + boundaryMixed + "\r\n"
	boundaryMixedEnd         = "--" + boundaryMixed + "--\r\n"
	boundaryHTMLRelated      = "===============_HTML_RELATED=="
	boundaryHTMLRelatedBegin = "--" + boundaryHTMLRelated + "\r\n"
	boundaryHTMLRelatedEnd   = "--" + boundaryHTMLRelated + "--\r\n"
	boundaryAMPRelated       = "===============_AMP_RELATED=="
	boundaryAMPRelatedBegin  = "--" + boundaryAMPRelated + "\r\n"
	boundaryAMPRelatedEnd    = "--" + boundaryAMPRelated + "--\r\n"
	boundaryAlternative      = "===============_ALTERNATIVE=="
	boundaryAlternativeBegin = "--" + boundaryAlternative + "\r\n"
	boundaryAlternativeEnd   = "--" + boundaryAlternative + "--\r\n"
)

// Builder helper for create email
type Builder struct {
	From             string
	To               string
	Subject          string
	subjectFunc      func(io.Writer) error
	replyTo          string
	headers          []string
	mimeHeader       textproto.MIMEHeader
	htmlPart         []byte
	textPart         []byte
	ampPart          []byte
	htmlFunc         func(io.Writer) error
	textFunc         func(io.Writer) error
	ampFunc          func(io.Writer) error
	htmlRelatedFiles []*os.File
	ampRelatedFiles  []*os.File
	attachments      []*os.File
	dkim             builderDKIM
}

type builderDKIM struct {
	domain     string
	selector   string
	privateKey []byte
}

// SetDKIM sign DKIM parameters
func (b *Builder) SetDKIM(domain, selector string, privateKey []byte) {
	b.dkim.domain = domain
	b.dkim.selector = selector
	b.dkim.privateKey = privateKey
}

// SetFrom email sender
func (b *Builder) SetFrom(name, email string) {
	b.From = mime.BEncoding.Encode("utf-8", name) + "<" + email + ">"
}

// SetTo email recipient
func (b *Builder) SetTo(name, email string) {
	b.To = mime.BEncoding.Encode("utf-8", name) + "<" + email + ">"
}

// SetSubject set email subject
func (b *Builder) SetSubject(text string) {
	b.Subject = text
}

// AddSubjectFunc add writer function for subject
func (b *Builder) AddSubjectFunc(f func(io.Writer) error) {
	b.subjectFunc = f
}

// AddReplyTo add Reply-To header
func (b *Builder) AddReplyTo(name, email string) {
	b.replyTo = email
}

// AddHTMLFunc add writer function for HTML
func (b *Builder) AddHTMLFunc(f func(io.Writer) error, file ...string) error {
	for i := range file {
		file, err := os.Open(file[i])
		if err != nil {
			return err
		}
		b.htmlRelatedFiles = append(b.htmlRelatedFiles, file)
	}
	b.htmlFunc = f
	return nil
}

// AddTextFunc add writer function for plain text
func (b *Builder) AddTextFunc(f func(io.Writer) error) {
	b.textFunc = f
}

// AddAMPFunc add writer function for AMP HTML
func (b *Builder) AddAMPFunc(f func(io.Writer) error, file ...string) error {
	for i := range file {
		file, err := os.Open(file[i])
		if err != nil {
			return err
		}
		b.ampRelatedFiles = append(b.ampRelatedFiles, file)
	}
	b.ampFunc = f
	return nil
}

// AddHeader add extra header to email
func (b *Builder) AddHeader(headers ...string) {
	for i := range headers {
		b.headers = append(b.headers, headers[i]+"\r\n")
	}
}

// AddMIMEHeader add extra mime header to email
func (b *Builder) AddMIMEHeader(mimeHeader textproto.MIMEHeader) {
	b.mimeHeader = mimeHeader
}

// AddHTMLPart add text/html content with related file.
//
// Example use related file in html
//  AddHTMLPart(
//  	`... <img src="cid:myImage.jpg" width="500px" height="250px" border="1px" alt="My image"/> ...`,
//  	"/path/to/attach/myImage.jpg",
//  )
func (b *Builder) AddHTMLPart(html []byte, file ...string) (err error) {
	for i := range file {
		file, err := os.Open(file[i])
		if err != nil {
			return err
		}
		b.htmlRelatedFiles = append(b.htmlRelatedFiles, file)
	}
	b.htmlPart = html
	return nil
}

// AddTextHTML
// Deprecated: use AddHTMLPart
func (b *Builder) AddTextHTML(html []byte, file ...string) (err error) {
	return b.AddHTMLPart(html, file...)
}

// AddTextPart add plain text
func (b *Builder) AddTextPart(text []byte) {
	b.textPart = text
}

// AddTextPlain add plain text
// Deprecated: use AddTextPart
func (b *Builder) AddTextPlain(text []byte) {
	b.AddTextPart(text)
}

// AddAMPPart add text/x-amp-html content with related file.
func (b *Builder) AddAMPPart(amp []byte, file ...string) (err error) {
	for i := range file {
		file, err := os.Open(file[i])
		if err != nil {
			return err
		}
		b.ampRelatedFiles = append(b.ampRelatedFiles, file)
	}
	b.ampPart = amp
	return nil
}

// AddAttachment add attachment files to email
func (b *Builder) AddAttachment(file ...string) error {
	for i := range file {
		file, err := os.Open(file[i])
		if err != nil {
			return err
		}
		b.attachments = append(b.attachments, file)
	}
	return nil
}

// Email return Email struct with render function
func (b *Builder) Email(id string, resultFunc func(Result)) Email {
	email := new(Email)
	email.ID = id
	email.From = b.From
	email.To = b.To
	email.ResultFunc = resultFunc
	email.WriteCloser = func(w io.WriteCloser) (err error) {
		defer w.Close()
		if b.dkim.domain == "" {
			err = b.headersBuilder(w)
			if err != nil {
				return err
			}
			err = b.bodyBuilder(w)
			return err
		}

		block, _ := pem.Decode(b.dkim.privateKey)
		if block == nil {
			return errors.New("dkim: cannot decode key")
		}
		var privateKey crypto.Signer
		switch strings.ToUpper(block.Type) {
		case "RSA PRIVATE KEY":
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return fmt.Errorf("error RSA private key: '%s'", err)
			}
		case "EDDSA PRIVATE KEY":
			if len(block.Bytes) != ed25519.PrivateKeySize {
				return fmt.Errorf("invalid Ed25519 private key size")
			}
			privateKey = ed25519.PrivateKey(block.Bytes)
		default:
			return fmt.Errorf("unknown private key type: '%v'", block.Type)
		}
		signer, err := dkim.NewSigner(&dkim.SignOptions{
			Domain:   b.dkim.domain,
			Selector: b.dkim.selector,
			HeaderKeys: []string{
				"From",
				"Subject",
				"To",
			},
			Signer:   privateKey,
		})
		if err := b.headersBuilder(signer); err != nil {
			return err
		}
		if err := b.bodyBuilder(signer);  err != nil {
			return err
		}
		if err := signer.Close(); err != nil {
			return err
		}
		if _, err := w.Write([]byte(signer.Signature())); err != nil {
			return err
		}

		if err := b.headersBuilder(w); err != nil {
			return err
		}
		return b.bodyBuilder(w)
	}
	return *email
}

func (b *Builder) headersBuilder(w io.Writer) error {
	err := b.writeHeaders(w)
	if err != nil {
		return err
	}

	switch {
	case b.isMultipart():
		err = b.writeMultipartHeader(w)
	case b.hasAlternative():
		err = b.writeAlternativeHeader(w)
	case b.hasText():
		err = b.writeTextPartHeader(w)
	case b.hasAMP():
		err = b.writeAMPPartHeader(w)
	case b.hasHTML():
		err = b.writeHTMLPartHeader(w)
	case b.hasAttachment():
		err = b.writeMultipartHeader(w)
	}
	if err != nil {
		return err
	}

	return err
}

func (b *Builder) bodyBuilder(w io.Writer) error {
	switch {
	case b.isMultipart() || b.hasAttachment():
		return b.multipartBuilder(w)
	case b.hasAlternative():
		return b.alternativeBuilder(w)
	case b.hasText():
		return b.writeTextPart(w)
	case b.hasAMP():
		return b.writeAMPPart(w)
	case b.hasHTML():
		return b.writeHTMLPart(w)
	}
	return nil
}

func (b *Builder) multipartBuilder(w io.Writer) error {
	switch {
	case b.hasAlternative():
		if _, err := w.Write([]byte(boundaryMixedBegin)); err != nil {
			return err
		}
		if err := b.writeAlternativeHeader(w); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
		if err := b.alternativeBuilder(w); err != nil {
			return err
		}
	case b.hasText():
		if _, err := w.Write([]byte(boundaryMixedBegin)); err != nil {
			return err
		}
		if err := b.writeTextPartHeader(w); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
		if err := b.writeTextPart(w); err != nil {
			return err
		}
	case b.hasAMP():
		if _, err := w.Write([]byte(boundaryMixedBegin)); err != nil {
			return err
		}
		if err := b.writeAMPPartHeader(w); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
		if err := b.writeAMPPart(w); err != nil {
			return err
		}
	case b.hasHTML():
		if _, err := w.Write([]byte(boundaryMixedBegin)); err != nil {
			return err
		}
		if err := b.writeHTMLPartHeader(w); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
		if err := b.writeHTMLPart(w); err != nil {
			return err
		}
	}

	// Attachments
	if err := b.writeAttachment(w); err != nil {
		return err
	}
	if _, err := w.Write([]byte(boundaryMixedEnd)); err != nil {
		return err
	}

	return nil
}

func (b *Builder) alternativeBuilder(w io.Writer) error {
	if b.hasText() {
		if _, err := w.Write([]byte(boundaryAlternativeBegin)); err != nil {
			return err
		}
		if err := b.writeTextPartHeader(w); err != nil {
			return err
		}
		if err := b.writeTextPart(w); err != nil {
			return err
		}
	}

	if b.hasAMP() {
		if _, err := w.Write([]byte(boundaryAlternativeBegin)); err != nil {
			return err
		}
		if err := b.writeAMPPartHeader(w); err != nil {
			return err
		}
		if err := b.writeAMPPart(w); err != nil {
			return err
		}
	}

	if b.hasHTML() {
		if _, err := w.Write([]byte(boundaryAlternativeBegin)); err != nil {
			return err
		}
		if err := b.writeHTMLPartHeader(w); err != nil {
			return err
		}
		if err := b.writeHTMLPart(w); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(boundaryAlternativeEnd)); err != nil {
		return err
	}

	_, err := w.Write([]byte("\r\n"))
	return err
}

func (b *Builder) writeMultipartHeader(w io.Writer) error {
	_, err := w.Write([]byte("Content-Type: multipart/mixed; boundary=\"" + boundaryMixed + "\"\r\n\r\n"))
	return err
}

func (b *Builder) writeAlternativeHeader(w io.Writer) error {
	_, err := w.Write([]byte("Content-Type: multipart/alternative; boundary=\"" + boundaryAlternative + "\"\r\n\r\n"))
	return err
}

func (b *Builder) writeTextPartHeader(w io.Writer) error {
	_, err := w.Write([]byte("Content-Type: text/plain; charset=\"utf-8\"\r\nContent-Transfer-Encoding: base64\r\n\r\n"))
	return err
}

func (b *Builder) writeAMPPartHeader(w io.Writer) error {
	if b.hasAMPRelated() {
		if _, err := w.Write([]byte("Content-Type: multipart/related; boundary=\"" + boundaryAMPRelated + "\"\r\n\r\n")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(boundaryAMPRelatedBegin)); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("Content-Type: text/x-amp-html; charset=\"utf-8\"\r\nContent-Transfer-Encoding: base64\r\n\r\n"))
	return err
}

func (b *Builder) writeHTMLPartHeader(w io.Writer) error {
	if b.hasHTMLRelated() {
		if _, err := w.Write([]byte("Content-Type: multipart/related; boundary=\"" + boundaryHTMLRelated + "\"\r\n\r\n")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(boundaryHTMLRelatedBegin)); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("Content-Type: text/html; charset=\"utf-8\"\r\nContent-Transfer-Encoding: base64\r\n\r\n"))
	return err
}

func (b *Builder) writeHeaders(w io.Writer) error {
	if _, err := w.Write([]byte("From: " + b.From + "\r\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("To: " + b.To + "\r\n")); err != nil {
		return err
	}
	if b.replyTo != "" {
		if _, err := w.Write([]byte("Reply-To: <" + b.replyTo + ">\r\n")); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte("MIME-Version: 1.0\r\n")); err != nil {
		return err
	}
	for i := range b.headers {
		if _, err := w.Write([]byte(b.headers[i])); err != nil {
			return err
		}
	}
	for k, v := range b.mimeHeader {
		if _, err := w.Write([]byte(k + ": ")); err != nil {
			return err
		}
		for i := range v {
			if len(v) == (i + 1) {
				if _, err := w.Write([]byte(" " + v[i])); err != nil {
					return err
				}
			} else {
				if _, err := w.Write([]byte(" " + v[i] + ";\r\n\t")); err != nil {
					return err
				}
			}
		}
		if _, err := w.Write([]byte("\r\n")); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("Subject: ")); err != nil {
		return err
	}
	subj, err := b.makeSubject()
	if err != nil {
		return err
	}
	_, err = w.Write(subj)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\r\n"))
	return err
}

func (b *Builder) makeSubject() ([]byte, error) {
	var err error
	subj := bytes.NewBufferString(b.Subject)
	if b.subjectFunc != nil {
		err = b.subjectFunc(subj)
		if err != nil {
			return nil, err
		}
	}
	return []byte(mime.BEncoding.Encode("utf-8", subj.String())), nil
}

// Text part
func (b *Builder) writeTextPart(w io.Writer) error {
	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 76) // 76 from RFC
	b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)
	defer b64Enc.Close()

	if _, err := b64Enc.Write(b.textPart); err != nil {
		return err
	}

	if b.textFunc != nil {
		if err := b.textFunc(b64Enc); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("\r\n\r\n"))
	return err
}

// HTML part
func (b *Builder) writeHTMLPart(w io.Writer) error {
	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 76) // 76 from RFC
	b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)
	defer b64Enc.Close()

	if _, err := b64Enc.Write(b.htmlPart); err != nil {
		return err
	}

	if b.htmlFunc != nil {
		if err := b.htmlFunc(b64Enc); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
		return err
	}

	// related files
	for i := range b.htmlRelatedFiles {
		if _, err := w.Write([]byte("\r\n")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(boundaryHTMLRelatedBegin)); err != nil {
			return err
		}

		if err := fileBase64Writer(w, b.htmlRelatedFiles[i], "inline"); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
	}
	if b.hasHTMLRelated() {
		if _, err := w.Write([]byte(boundaryHTMLRelatedEnd)); err != nil {
			return err
		}
	}

	return nil
}

// AMP part
func (b *Builder) writeAMPPart(w io.Writer) error {
	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 76) // 76 from RFC
	b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)
	defer b64Enc.Close()

	if _, err := b64Enc.Write(b.ampPart); err != nil {
		return err
	}

	if b.ampFunc != nil {
		if err := b.ampFunc(b64Enc); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
		return err
	}

	// related files
	for i := range b.ampRelatedFiles {
		if _, err := w.Write([]byte("\r\n")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(boundaryAMPRelatedBegin)); err != nil {
			return err
		}

		if err := fileBase64Writer(w, b.ampRelatedFiles[i], "inline"); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
	}

	if b.hasAMPRelated() {
		if _, err := w.Write([]byte(boundaryAMPRelatedEnd)); err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) writeAttachment(w io.Writer) error {
	for i := range b.attachments {
		if _, err := w.Write([]byte(boundaryMixedBegin)); err != nil {
			return err
		}
		if err := fileBase64Writer(w, b.attachments[i], "attachment"); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\r\n\r\n")); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) hasText() bool {
	return len(b.textPart) != 0 || b.textFunc != nil
}

func (b *Builder) hasHTML() bool {
	return len(b.htmlPart) != 0 || b.htmlFunc != nil
}

func (b *Builder) hasAMP() bool {
	return len(b.ampPart) != 0 || b.ampFunc != nil
}

func (b *Builder) hasAlternative() bool {
	var c = 0
	if b.hasText() {
		c++
	}
	if b.hasHTML() {
		c++
	}
	if b.hasAMP() {
		c++
	}
	return c > 1
}

func (b *Builder) hasHTMLRelated() bool {
	return b.hasHTML() && (len(b.htmlRelatedFiles) > 0)
}

func (b *Builder) hasAMPRelated() bool {
	return b.hasAMP() && (len(b.ampRelatedFiles) > 0)
}

func (b *Builder) hasAttachment() bool {
	return len(b.attachments) > 0
}

func (b *Builder) isMultipart() bool {
	var c = 0
	if b.hasText() || b.hasAMP() || b.hasHTML() {
		c++
	}

	c = c + len(b.attachments)
	return c > 1
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
			_, err = w.writer.Write(w.dr)
			if err != nil {
				break
			}
		}
	}
	return w.n, err
}

func fileBase64Writer(w io.Writer, f *os.File, disposition string) error {
	var err error
	var info os.FileInfo
	name := filepath.Base(f.Name())
	info, err = f.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	buf := make([]byte, 512)
	_, err = f.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	content := http.DetectContentType(buf)
	_, err = f.Seek(0, 0)
	var contentID string
	if disposition == "inline" {
		contentID = "Content-ID: <" + name + ">\r\n"
	}
	_, err = w.Write([]byte(fmt.Sprintf(
		"Content-Type: %s; name=\"%s\"\r\nContent-Transfer-Encoding: base64\r\n%sContent-Disposition: %s; filename=\"%s\"; size=%d;\r\n\r\n",
		content,
		name,
		contentID,
		disposition,
		name,
		size)))
	if err != nil {
		return err
	}

	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 76) // 76 from RFC
	b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)

	if _, err := io.Copy(b64Enc, f); err != nil {
		return err
	}

	return b64Enc.Close()
}
