package smtpSender

import (
	"bytes"
	"encoding/base64"
	tmplHTML "html/template"
	"io"
	"io/ioutil"
	"net/textproto"
	"testing"
	tmplText "text/template"
)

var (
	pkey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQCvis6cltd3R1Y2xJm1mUpR9aBnpH2x8qux8kg1Pvsk9reeXN+n
wGbIISC/yM/hvQx9lCQki/OETKQYdTERjndtaJv1AgRhMccJNocXheWJD9dMUMYd
PX2PMLkFTguUT47bPt9rR9wXCDyf1tBIDnFzy8aaReLIupuDhV0al7mWOQIDAQAB
AoGAOXcJN/2xP1zc/kTRxL8Ps1DjV8pjU3OLfU9BEB0z/d++MFta5AF6JB2kKORG
GTHX+uwaANTHvRGRzmfezk6DDYUD0I0v8kdfp1EHX1klMHHu8jQFA7mkoBbUwjca
oqwizZNUkXRNV2V6E5U933+TBFm0J0ejF0vnUDD1dpvbF+ECQQDb+D67wI39KzVb
VOs49RDEXYVLihmXsWZopbuoMaS5ZQ8q7cZ+qDIJimxTvObGQhO57EgTe2M9IZKh
v0RsV15lAkEAzEuhpKUnJZkxtY154SiU2QzbGm+W70YQovqfZYBHFyLVAHfmHjNN
LgbWeHA63ata28rkfe6m9sBFIR7teEfBRQJAZLkKOLyWB7wWRYjf4IfOsqvEEm/d
AinYI8jn4b9BlybgSB7yiiKILvg0XC+eWF//Wl4ILuuL6H0MAIZtVVK4RQJBAIyw
GOUVhtvxn7Xzc9eG5tqCa/DMoBivG43hIhv4NvzL0/u6lhJ+KcxkkRXn0+ILu0pZ
cvj2fKy4w+KHNen7IDECQQCv4zeGpyO2AZnEBeHqHs9PCySglqIiHc56l9fSZu0u
6nAfGefm766gqBoYTC/1upkMYJpxizyH7U/7WessATfA
-----END RSA PRIVATE KEY-----`)
	textPart                = []byte("Привет, буфет\r\nЗдорова, колбаса!\r\nКак твои дела?\r\n0123456789\r\nabcdefgh\r\n")
	htmlPart                = []byte("<h1>Привет, буфет</h1><br/>\r\n<h2>Здорова, колбаса!</h2><br/>\r\n<h3>Как твои дела?</h3><br/>\r\n0123456789\r\nabcdefgh\r\n")
	ampPart                 = []byte(`<!doctype html>\r\n<html amp4email>\r\n<head>\r\n<title>Hello World</title>\r\n<meta charset=\"utf-8\">\r\n<style amp4email-boilerplate>body{visibility:hidden}</style>\r\n<script async src=\"https://cdn.ampproject.org/v0.js\"></script>\r\n<script async custom-element=\"amp-carousel\" src=\"https://cdn.ampproject.org/v0/amp-carousel-0.1.js\"></script>\r\n</head>\r\n<body>\r\n<p>Hello World</p>\r\n<amp-carousel width=\"400\" height=\"300\" layout=\"responsive\" type=\"slides\">\r\n  <amp-img src=\"https://loremflickr.com/400/300?random=1\" width=\"400\" height=\"300\" layout=\"responsive\" alt=\"\"></amp-img>\r\n  <amp-img src=\"https://loremflickr.com/400/300?random=2\" width=\"400\" height=\"300\" layout=\"responsive\" alt=\"\"></amp-img>\r\n</amp-carousel>\r\n</body>\r\n</html>`)
	discard  io.WriteCloser = devNull{}
)

type devNull struct{}

func (devNull) Write(p []byte) (int, error) { return len(p), nil }
func (devNull) Close() error                { return nil }

func TestBuilder(t *testing.T) {
	bldr := new(Builder)
	bldr.SetSubject("Test subject")
	bldr.SetFrom("Вася", "vasya@mail.tld")
	bldr.SetTo("Петя", "petya@mail.tld")

	bldr.AddHeader("Message-ID: <test_message>")
	mimeHeader := textproto.MIMEHeader{}
	mimeHeader.Add("Content-Language", "ru")
	mimeHeader.Add("Precedence", " bulk")
	bldr.AddMIMEHeader(mimeHeader)

	bldr.AddTextPart(textPart)
	bldr.AddAMPPart(ampPart)
	bldr.AddHTMLPart(htmlPart, "./testdata/prwoman.png")
	bldr.AddAttachment("./testdata/knwoman.png")

	//_ = bldr.Email("Id-123", func(Result) {})
	email := bldr.Email("Id-123", func(Result) {})
	err := email.WriteCloser(discard)
	if err != nil {
		t.Error(err)
	}
}

func TestBuilderTemplate(t *testing.T) {
	bldr := new(Builder)
	data := map[string]string{"Name": "Вася"}

	subj := tmplText.New("Text")
	subj.Parse("Test subject for {{.Name}}")
	bldr.AddSubjectFunc(func(w io.Writer) error {
		return subj.Execute(w, data)
	})

	bldr.SetFrom("Вася", "vasya@mail.tld")
	bldr.SetTo("Петя", "petya@mail.tld")

	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")

	html := tmplHTML.New("HTML")
	html.Parse(`<h1>This 'HTML' template.</h1><h2>Hello {{.Name}}</h2>`)
	text := tmplText.New("Text")
	text.Parse("This 'Text' template. Hello {{.Name}}")

	bldr.AddTextFunc(func(w io.Writer) error {
		return text.Execute(w, data)
	})
	bldr.AddHTMLFunc(func(w io.Writer) error {
		return html.Execute(w, data)
	})

	//_ = bldr.Email("Id-123", func(Result) {})
	email := bldr.Email("Id-123", func(Result) {})
	err := email.WriteCloser(discard)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkBuilder(b *testing.B) {
	bldr := new(Builder)
	bldr.SetSubject("Test subject")
	bldr.SetFrom("Вася", "vasya@mail.tld")
	bldr.SetTo("Петя", "petya@mail.tld")
	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")
	bldr.AddTextPart(textPart)
	bldr.AddHTMLPart(htmlPart)
	var err error
	for n := 0; n < b.N; n++ {
		//_ = bldr.Email("Id-123", func(Result) {})
		email := bldr.Email("Id-123", func(Result) {})
		err = email.WriteCloser(discard)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkBuilderTemplate(b *testing.B) {
	bldr := new(Builder)
	data := map[string]string{"Name": "Вася"}

	subj := tmplText.New("Text")
	subj.Parse("Test subject for {{.Name}}")
	bldr.AddSubjectFunc(func(w io.Writer) error {
		return subj.Execute(w, data)
	})

	bldr.SetFrom("Вася", "vasya@mail.tld")
	bldr.SetTo("Петя", "petya@mail.tld")

	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")

	html := tmplHTML.New("HTML")
	html.Parse(`<h1>This 'HTML' template.</h1><h2>Hello {{.Name}}</h2>`)
	text := tmplText.New("Text")
	text.Parse("This 'Text' template. Hello {{.Name}}")

	bldr.AddTextFunc(func(w io.Writer) error {
		return text.Execute(w, data)
	})
	bldr.AddHTMLFunc(func(w io.Writer) error {
		return html.Execute(w, data)
	})

	var err error
	for n := 0; n < b.N; n++ {
		//_ = bldr.Email("Id-123", func(Result) {})
		email := bldr.Email("Id-123", func(Result) {})
		err = email.WriteCloser(discard)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkBuilderAttachment(b *testing.B) {
	bldr := new(Builder)
	bldr.SetSubject("Test subject")
	bldr.SetFrom("Вася", "vasya@mail.tld")
	bldr.SetTo("Петя", "petya@mail.tld")
	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")
	bldr.AddTextPart(textPart)
	bldr.AddHTMLPart(htmlPart, "./testdata/prwoman.png")
	bldr.AddAttachment("./testdata/knwoman.png")
	var err error
	for n := 0; n < b.N; n++ {
		//_ = bldr.Email("Id-123", func(Result) {})
		email := bldr.Email("Id-123", func(Result) {})
		err = email.WriteCloser(discard)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkBuilderDKIM(b *testing.B) {
	bldr := new(Builder)
	bldr.SetDKIM("mail.ru", "test", pkey)
	bldr.SetSubject("Test subject")
	bldr.SetFrom("Вася", "vasya@mail.tld")
	bldr.SetTo("Петя", "petya@mail.tld")
	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")
	bldr.AddTextPart(textPart)
	bldr.AddHTMLPart(htmlPart)
	var err error
	for n := 0; n < b.N; n++ {
		//_ = bldr.Email("Id-123", func(Result) {})
		email := bldr.Email("Id-123", func(Result) {})
		err = email.WriteCloser(discard)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkBuilderAttachmentDKIM(b *testing.B) {
	bldr := new(Builder)
	bldr.SetDKIM("mail.ru", "test", pkey)
	bldr.SetSubject("Test subject")
	bldr.SetFrom("Вася", "vasya@mail.tld")
	bldr.SetTo("Петя", "petya@mail.tld")
	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")
	bldr.AddTextPart(textPart)
	bldr.AddHTMLPart(htmlPart, "./testdata/prwoman.png")
	bldr.AddAttachment("./testdata/knwoman.png")
	var err error
	for n := 0; n < b.N; n++ {
		_ = bldr.Email("Id-123", func(Result) {})
		email := bldr.Email("Id-123", func(Result) {})
		err = email.WriteCloser(discard)
		if err != nil {
			b.Error(err)
		}
	}
}

func TestDelimitWriter(t *testing.T) {
	m := []byte(htmlPart)
	w := &bytes.Buffer{}
	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 16)
	encoder := base64.NewEncoder(base64.StdEncoding, dwr)
	_, err := encoder.Write(m)
	if err != nil {
		t.Error(err)
	}
	err = encoder.Close()
	if err != nil {
		t.Error(err)
	}

	d, _ := base64.StdEncoding.DecodeString(w.String())
	if c := bytes.Compare(m, d); c != 0 {
		t.Error("Base64 encode/decode not equivalent")
	}
}

func BenchmarkBase64DelimitWriter(b *testing.B) {
	m := []byte("<h1>Hello, буфет</h1><br/>\r\n<h2>Здорова, колбаса!</h2><br/>\r\n<h3>Как твои дела?</h3><br/>\r\n0123456789\r\nabcdefgh\r\n")
	w := ioutil.Discard
	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 8)
	encoder := base64.NewEncoder(base64.StdEncoding, dwr)
	for n := 0; n < b.N; n++ {
		_, err := encoder.Write(m)
		if err != nil {
			b.Error(err)
		}
		err = encoder.Close()
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkDelimitWriter(b *testing.B) {
	m := []byte("<h1>Hello, буфет</h1><br/>\r\n<h2>Здорова, колбаса!</h2><br/>\r\n<h3>Как твои дела?</h3><br/>\r\n0123456789\r\nabcdefgh\r\n")
	w := ioutil.Discard
	dwr := newDelimitWriter(w, []byte{0x0d, 0x0a}, 8)
	for n := 0; n < b.N; n++ {
		_, err := dwr.Write(m)
		if err != nil {
			b.Error(err)
		}
	}
}
