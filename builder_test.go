package smtpSender

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"testing"
)

var (
	textPlain = []byte("Привет, буфет\r\nЗдорова, колбаса!\r\nКак твои дела?\r\n0123456789\r\nabcdefgh\r\n")
	textHTML  = []byte("<h1>Привет, буфет</h1><br/>\r\n<h2>Здорова, колбаса!</h2><br/>\r\n<h3>Как твои дела?</h3><br/>\r\n0123456789\r\nabcdefgh\r\n")
)

func TestBuilder(t *testing.T) {
	bldr := new(Builder)
	bldr.SetSubject("Test subject")
	bldr.SetFrom("Вася", "vasya@mail.ru")
	bldr.SetTo("Петя", "petya@mail.ru")
	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")
	bldr.AddTextPlain(textPlain)
	bldr.AddTextHTML(textHTML, "./sender.go", "./email.go")
	bldr.AddAttachment("./connect.go")
	w := &bytes.Buffer{}
	email := bldr.Email("Id-123", func(Result) {})
	err := email.WriteCloser(w)
	if err != nil {
		t.Error(err)
	}
	//print(w.String())
}

func BenchmarkBuilder(b *testing.B) {
	bldr := new(Builder)
	bldr.SetSubject("Test subject")
	bldr.SetFrom("Вася", "vasya@mail.ru")
	bldr.SetTo("Петя", "petya@mail.ru")
	bldr.AddHeader("Content-Language: ru", "Message-ID: <test_message>", "Precedence: bulk")
	bldr.AddTextPlain(textPlain)
	bldr.AddTextHTML(textHTML, "./sender.go", "./email.go")
	bldr.AddAttachment("./connect.go")
	var err error
	w := ioutil.Discard
	for n := 0; n < b.N; n++ {
		email := bldr.Email("Id-123", func(Result) {})
		err = email.WriteCloser(w)
		if err != nil {
			b.Error(err)
		}
	}
}

func TestDelimitWriter(t *testing.T) {
	m := []byte(textHTML)
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
