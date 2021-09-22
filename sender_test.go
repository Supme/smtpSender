package smtpSender_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/jhillyerd/enmime"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Supme/smtpSender"

	"github.com/chrj/smtpd"
)

const (
	testText = `Test text message.
Hello, World!`
	testHtml = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <title>Test content</title>
</head>
<body>
    <h2>Test HTML message</h2>
</body>`
	testAmp = `<!DOCTYPE html>
<html amp4email data-css-strict>
<head>
    <meta charset="utf-8" />
    <script async src="https://cdn.ampproject.org/v0.js"></script>
    <title>Test content</title>
</head>
<body>
    <h2>Test AMP message</h2>
</body>`
)

func TestEmail_Send(t *testing.T) {
	// start test smtp server
	received := make(chan receiveMail, 1)
	addr, closer := runsslserver(
		t,
		&smtpd.Server{
			//ProtocolLogger: log.New(os.Stdout, "smtpdLog: ", log.Lshortfile),
		},
		received)
	defer closer()
	//t.Logf("Listen: %s", addr)

	testEmails := map[string]testEmail{
		"all parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "🚀 Test message 🚀",
			text:           testText,
			html:           testHtml,
			htmlRelated: []string{"testdata/knwoman.png", "testdata/prwoman.png"},
			amp:            testAmp,
			attachments:    []string{"testdata/knwoman.png", "testdata/prwoman.png"},
		},
		"all parts without html related": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			text:           testText,
			html:           testHtml,
			htmlRelated: []string{"testdata/knwoman.png", "testdata/prwoman.png"},
			amp:            testAmp,
			attachments:    []string{"testdata/knwoman.png", "testdata/prwoman.png"},
		},

		"text, html and amp parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			text:           testText,
			html:           testHtml,
			amp:            testAmp,
		},

		"text and html parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			text:           testText,
			html:           testHtml,
		},

		"only text part": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Тестовое сообщение",
			// one part always add new line
			text: testText + "\n",
		},

		"only html part": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			// one part always add new line
			html: testHtml + "\n",
		},

		"html part with related": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			// one part always add new line
			html: testHtml + "\n",
			htmlRelated: []string{"testdata/knwoman.png", "testdata/prwoman.png"},
		},

		// this unreal but... need add AMP support to github.com/jhillyerd/enmime
		//"only amp part": {
		//	heloName:       "localtest",
		//	senderName:     "Sender Отправитель",
		//	senderEmail:    "sender@localhost.localdomain",
		//	recipientName:  "Recipient Получатель",
		//	recipientEmail: "recipient@linklocal.supme.ru",
		//	subject:        "Test message",
		//	// one part always add new line
		//	amp:           testAmp+"\n",
		//},

		"html and two attachments parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			html:           testHtml,
			attachments:    []string{"testdata/knwoman.png", "testdata/prwoman.png"},
		},

		"html and one attachments parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			html:           testHtml,
			attachments:    []string{"testdata/knwoman.png"},
		},

		"amp and one attachments parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			amp:            testAmp,
			attachments:    []string{"testdata/knwoman.png"},
		},

		"text and one attachments parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			text:           testText,
			attachments:    []string{"testdata/knwoman.png"},
		},

		"only attachments parts": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			attachments:    []string{"testdata/knwoman.png", "testdata/prwoman.png"},
		},

		"one attachment part": {
			heloName:       "localtest",
			senderName:     "Sender Отправитель",
			senderEmail:    "sender@localhost.localdomain",
			recipientName:  "Recipient Получатель",
			recipientEmail: "recipient@linklocal.supme.ru",
			subject:        "Test message",
			attachments:    []string{"testdata/prwoman.png"},
		},
	}

	for id := range testEmails {
		testEmails[id].start(t, addr, received, id)
	}
}

type testEmail struct {
	heloName                      string
	senderName, senderEmail       string
	recipientName, recipientEmail string
	subject                       string
	text                          string
	html                          string
	htmlRelated                   []string
	amp                           string
	attachments                   []string
}

func (te testEmail) start(t *testing.T, addr string, received chan receiveMail, id string) {
	// build email
	b := smtpSender.NewBuilder().
		SetFrom(te.senderName, te.senderEmail).
		SetTo(te.recipientName, te.recipientEmail).
		SetSubject(te.subject)
	_ = b.AddTextPart([]byte(te.text))
	_ = b.AddHTMLPart([]byte(te.html), te.htmlRelated...)
	_ = b.AddAMPPart([]byte(te.amp))
	for i := range te.attachments {
		err := b.AddAttachment(te.attachments[i])
		if err != nil {
			t.Errorf("add attachment to email with id '%s': %s", id, err)
		}
	}

	// send email
	e := b.Email(id,
		func(result smtpSender.Result) {
			//t.Logf("%+v", result)
			if result.Err != nil {
				t.Error(result.Err)
			}
		})

	conn := new(smtpSender.Connect)
	conn.SetHostName(te.heloName)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Errorf("split hostport: %v", err)
	}
	p, _ := strconv.Atoi(port)
	conn.SetSMTPport(p)

	e.Send(conn, nil)

	// receive email
	timeout := time.NewTimer(time.Second)
	select {
	case <-timeout.C:
		t.Errorf("timeout receive email id '%s'", id)
	case r := <-received:
		//t.Logf("receive email with id '%s'", id)

		// check helo name
		//t.Logf("HeloName: '%s'\r", r.HeloName)
		if r.HeloName != te.heloName {
			t.Errorf("compare helo name '%s' fail", te.heloName)
		}

		// check sender
		//t.Logf("Sender: '%s'", r.Sender)
		if r.Sender != te.senderEmail {
			t.Errorf("compare sender email '%s' fail", te.senderEmail)
		}

		// check recipient
		//t.Logf("Recipient: '%s'", r.Recipients)
		if len(r.Recipients) != 1 {
			t.Errorf("recipients count '%d' in email id '%s' fail", len(r.Recipients), id)
		} else if r.Recipients[0] != te.recipientEmail {
			t.Errorf("compare recipient '%s' email id '%s' fail", te.recipientEmail, id)
		}

		//t.Logf("Data:\r\n%s", r.Data)
		buf := bytes.NewBuffer(r.Data)
		env, err := enmime.ReadEnvelope(buf)
		if err != nil {
			t.Errorf("parse MIME body error: %s", err)
		}

		// check subject
		//t.Logf("Subject: '%s'", env.GetHeader("Subject"))
		if env.GetHeader("Subject") != te.subject {
			t.Errorf("compare subject '%s' fail", te.subject)
		}

		// check text part
		if te.text != "" {
			//t.Logf("Text:\r\n'%s'", env.Text)
			if env.Text != te.text {
				t.Errorf("compare text part '%s' fail", env.Text)
			}
		}

		// check html part
		if te.html != "" {
			//t.Logf("HTML:\r\n'%s'", env.HTML)
			if env.HTML != te.html {
				t.Errorf("compare HTML part '%s' fail", env.HTML)
			}
		}

		// check amp part
		if te.amp != "" {
			var amp *enmime.Part
			for _, p := range env.OtherParts {
				if p.ContentType == "text/x-amp-html" {
					amp = p
					break
				}
			}
			if amp != nil {
				//t.Logf("AMP:\r\n'%s", string(amp.Content))
				if string(amp.Content) != te.amp {
					t.Errorf("compare AMP part '%s' fail", string(amp.Content))
				}
			} else {
				t.Errorf("amp part not found in email with id '%s'", id)
			}
		}

		// check html related attachments
		for i := range te.htmlRelated {
			var inline *enmime.Part
			for _, p := range env.Inlines {
				//t.Logf("html related '%s' file name '%v'", filepath.Base(te.htmlRelated[i]), p.FileName)
				if p.FileName == filepath.Base(te.htmlRelated[i]) {
					inline = p
					break
				}
			}
			if inline != nil {
				b, err := ioutil.ReadFile(te.htmlRelated[i])
				if err != nil {
					fmt.Print(err)
				}
				//t.Logf("compare '%s' type '%s'", inline.FileName, inline.ContentType)
				if bytes.Compare(b, inline.Content) != 0 {
					t.Errorf("compare attachment '%s' in mail with id '%s' fail", te.attachments[i], id)
				}
			} else {
				t.Errorf("attachment '%s' part not found in email with id '%s'", te.attachments[i], id)
			}
		}

		// check attachments
		for i := range te.attachments {
			var attach *enmime.Part
			for _, p := range env.Attachments {
				//t.Logf("attachment '%s' file name '%v'", filepath.Base(te.attachments[i]), p.FileName)
				if p.FileName == filepath.Base(te.attachments[i]) {
					attach = p
					break
				}
			}
			if attach != nil {
				b, err := ioutil.ReadFile(te.attachments[i])
				if err != nil {
					fmt.Print(err)
				}
				//t.Logf("compare '%s' type '%s'", attach.FileName, attach.ContentType)
				if bytes.Compare(b, attach.Content) != 0 {
					t.Errorf("compare attachment '%s' in mail with id '%s' fail", te.attachments[i], id)
				}
			} else {
				t.Errorf("attachment '%s' part not found in email with id '%s'", te.attachments[i], id)
			}
		}
	}
}

var localhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIIFkzCCA3ugAwIBAgIUQvhoyGmvPHq8q6BHrygu4dPp0CkwDQYJKoZIhvcNAQEL
BQAwWTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MB4X
DTIwMDUyMTE2MzI1NVoXDTMwMDUxOTE2MzI1NVowWTELMAkGA1UEBhMCQVUxEzAR
BgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5
IEx0ZDESMBAGA1UEAwwJbG9jYWxob3N0MIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEAk773plyfK4u2uIIZ6H7vEnTb5qJT6R/KCY9yniRvCFV+jCrISAs9
0pgU+/P8iePnZRGbRCGGt1B+1/JAVLIYFZuawILHNs4yWKAwh0uNpR1Pec8v7vpq
NpdUzXKQKIqFynSkcLA8c2DOZwuhwVc8rZw50yY3r4i4Vxf0AARGXapnBfy6WerR
/6xT7y/OcK8+8aOirDQ9P6WlvZ0ynZKi5q2o1eEVypT2us9r+HsCYosKEEAnjzjJ
wP5rvredxUqb7OupIkgA4Nq80+4tqGGQfWetmoi3zXRhKpijKjgxBOYEqSUWm9ws
/aC91Iy5RawyTB0W064z75OgfuI5GwFUbyLD0YVN4DLSAI79GUfvc8NeLEXpQvYq
+f8P+O1Hbv2AQ28IdbyQrNefB+/WgjeTvXLploNlUihVhpmLpptqnauw/DY5Ix51
w60lHIZ6esNOmMQB+/z/IY5gpmuo66yH8aSCPSYBFxQebB7NMqYGOS9nXx62/Bn1
OUVXtdtrhfbbdQW6zMZjka0t8m83fnGw3ISyBK2NNnSzOgycu0ChsW6sk7lKyeWa
85eJGsQWIhkOeF9v9GAIH/qsrgVpToVC9Krbk+/gqYIYF330tHQrzp6M6LiG5OY1
P7grUBovN2ZFt10B97HxWKa2f/8t9sfHZuKbfLSFbDsyI2JyNDh+Vk0CAwEAAaNT
MFEwHQYDVR0OBBYEFOLdIQUr3gDQF5YBor75mlnCdKngMB8GA1UdIwQYMBaAFOLd
IQUr3gDQF5YBor75mlnCdKngMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEL
BQADggIBAGddhQMVMZ14TY7bU8CMuc9IrXUwxp59QfqpcXCA2pHc2VOWkylv2dH7
ta6KooPMKwJ61d+coYPK1zMUvNHHJCYVpVK0r+IGzs8mzg91JJpX2gV5moJqNXvd
Fy6heQJuAvzbb0Tfsv8KN7U8zg/ovpS7MbY+8mRJTQINn2pCzt2y2C7EftLK36x0
KeBWqyXofBJoMy03VfCRqQlWK7VPqxluAbkH+bzji1g/BTkoCKzOitAbjS5lT3sk
oCrF9N6AcjpFOH2ZZmTO4cZ6TSWfrb/9OWFXl0TNR9+x5c/bUEKoGeSMV1YT1SlK
TNFMUlq0sPRgaITotRdcptc045M6KF777QVbrYm/VH1T3pwPGYu2kUdYHcteyX9P
8aRG4xsPGQ6DD7YjBFsif2fxlR3nQ+J/l/+eXHO4C+eRbxi15Z2NjwVjYpxZlUOq
HD96v516JkMJ63awbY+HkYdEUBKqR55tzcvNWnnfiboVmIecjAjoV4zStwDIti9u
14IgdqqAbnx0ALbUWnvfFloLdCzPPQhgLHpTeRSEDPljJWX8rmy8iQtRb0FWYQ3z
A2wsUyutzK19nt4hjVrTX0At9ku3gMmViXFlbvyA1Y4TuhdUYqJauMBrWKl2ybDW
yhdKg/V3yTwgBUtb3QO4m1khNQjQLuPFVxULGEA38Y5dXSONsYnt
-----END CERTIFICATE-----`)

var localhostKey = []byte(`-----BEGIN PRIVATE KEY-----
MIIJQgIBADANBgkqhkiG9w0BAQEFAASCCSwwggkoAgEAAoICAQCTvvemXJ8ri7a4
ghnofu8SdNvmolPpH8oJj3KeJG8IVX6MKshICz3SmBT78/yJ4+dlEZtEIYa3UH7X
8kBUshgVm5rAgsc2zjJYoDCHS42lHU95zy/u+mo2l1TNcpAoioXKdKRwsDxzYM5n
C6HBVzytnDnTJjeviLhXF/QABEZdqmcF/LpZ6tH/rFPvL85wrz7xo6KsND0/paW9
nTKdkqLmrajV4RXKlPa6z2v4ewJiiwoQQCePOMnA/mu+t53FSpvs66kiSADg2rzT
7i2oYZB9Z62aiLfNdGEqmKMqODEE5gSpJRab3Cz9oL3UjLlFrDJMHRbTrjPvk6B+
4jkbAVRvIsPRhU3gMtIAjv0ZR+9zw14sRelC9ir5/w/47Udu/YBDbwh1vJCs158H
79aCN5O9cumWg2VSKFWGmYumm2qdq7D8NjkjHnXDrSUchnp6w06YxAH7/P8hjmCm
a6jrrIfxpII9JgEXFB5sHs0ypgY5L2dfHrb8GfU5RVe122uF9tt1BbrMxmORrS3y
bzd+cbDchLIErY02dLM6DJy7QKGxbqyTuUrJ5Zrzl4kaxBYiGQ54X2/0YAgf+qyu
BWlOhUL0qtuT7+CpghgXffS0dCvOnozouIbk5jU/uCtQGi83ZkW3XQH3sfFYprZ/
/y32x8dm4pt8tIVsOzIjYnI0OH5WTQIDAQABAoICADBPw788jje5CdivgjVKPHa2
i6mQ7wtN/8y8gWhA1aXN/wFqg+867c5NOJ9imvOj+GhOJ41RwTF0OuX2Kx8G1WVL
aoEEwoujRUdBqlyzUe/p87ELFMt6Svzq4yoDCiyXj0QyfAr1Ne8sepGrdgs4sXi7
mxT2bEMT2+Nuy7StsSyzqdiFWZJJfL2z5gZShZjHVTfCoFDbDCQh0F5+Zqyr5GS1
6H13ip6hs0RGyzGHV7JNcM77i3QDx8U57JWCiS6YRQBl1vqEvPTJ0fEi8v8aWBsJ
qfTcO+4M3jEFlGUb1ruZU3DT1d7FUljlFO3JzlOACTpmUK6LSiRPC64x3yZ7etYV
QGStTdjdJ5+nE3CPR/ig27JLrwvrpR6LUKs4Dg13g/cQmhpq30a4UxV+y8cOgR6g
13YFOtZto2xR+53aP6KMbWhmgMp21gqxS+b/5HoEfKCdRR1oLYTVdIxt4zuKlfQP
pTjyFDPA257VqYy+e+wB/0cFcPG4RaKONf9HShlWAulriS/QcoOlE/5xF74QnmTn
YAYNyfble/V2EZyd2doU7jJbhwWfWaXiCMOO8mJc+pGs4DsGsXvQmXlawyElNWes
wJfxsy4QOcMV54+R/wxB+5hxffUDxlRWUsqVN+p3/xc9fEuK+GzuH+BuI01YQsw/
laBzOTJthDbn6BCxdCeBAoIBAQDEO1hDM4ZZMYnErXWf/jik9EZFzOJFdz7g+eHm
YifFiKM09LYu4UNVY+Y1btHBLwhrDotpmHl/Zi3LYZQscWkrUbhXzPN6JIw98mZ/
tFzllI3Ioqf0HLrm1QpG2l7Xf8HT+d3atEOtgLQFYehjsFmmJtE1VsRWM1kySLlG
11bQkXAlv7ZQ13BodQ5kNM3KLvkGPxCNtC9VQx3Em+t/eIZOe0Nb2fpYzY/lH1mF
rFhj6xf+LFdMseebOCQT27bzzlDrvWobQSQHqflFkMj86q/8I8RUAPcRz5s43YdO
Q+Dx2uJQtNBAEQVoS9v1HgBg6LieDt0ZytDETR5G3028dyaxAoIBAQDAvxEwfQu2
TxpeYQltHU/xRz3blpazgkXT6W4OT43rYI0tqdLxIFRSTnZap9cjzCszH10KjAg5
AQDd7wN6l0mGg0iyL0xjWX0cT38+wiz0RdgeHTxRk208qTyw6Xuh3KX2yryHLtf5
s3z5zkTJmj7XXOC2OVsiQcIFPhVXO3d38rm0xvzT5FZQH3a5rkpks1mqTZ4dyvim
p6vey4ZXdUnROiNzqtqbgSLbyS7vKj5/fXbkgKh8GJLNV4LMD6jo2FRN/LsEZKes
pxWNMsHBkv5eRfHNBVZuUMKFenN6ojV2GFG7bvLYD8Z9sja8AuBCaMr1CgHD8kd5
+A5+53Iva8hdAoIBAFU+BlBi8IiMaXFjfIY80/RsHJ6zqtNMQqdORWBj4S0A9wzJ
BN8Ggc51MAqkEkAeI0UGM29yicza4SfJQqmvtmTYAgE6CcZUXAuI4he1jOk6CAFR
Dy6O0G33u5gdwjdQyy0/DK21wvR6xTjVWDL952Oy1wyZnX5oneWnC70HTDIcC6CK
UDN78tudhdvnyEF8+DZLbPBxhmI+Xo8KwFlGTOmIyDD9Vq/+0/RPEv9rZ5Y4CNsj
/eRWH+sgjyOFPUtZo3NUe+RM/s7JenxKsdSUSlB4ZQ+sv6cgDSi9qspH2E6Xq9ot
QY2jFztAQNOQ7c8rKQ+YG1nZ7ahoa6+Tz1wAUnECggEAFVTP/TLJmgqVG37XwTiu
QUCmKug2k3VGbxZ1dKX/Sd5soXIbA06VpmpClPPgTnjpCwZckK9AtbZTtzwdgXK+
02EyKW4soQ4lV33A0lxBB2O3cFXB+DE9tKnyKo4cfaRixbZYOQnJIzxnB2p5mGo2
rDT+NYyRdnAanePqDrZpGWBGhyhCkNzDZKimxhPw7cYflUZzyk5NSHxj/AtAOeuk
GMC7bbCp8u3Ows44IIXnVsq23sESZHF/xbP6qMTO574RTnQ66liNagEv1Gmaoea3
ug05nnwJvbm4XXdY0mijTAeS/BBiVeEhEYYoopQa556bX5UU7u+gU3JNgGPy8iaW
jQKCAQEAp16lci8FkF9rZXSf5/yOqAMhbBec1F/5X/NQ/gZNw9dDG0AEkBOJQpfX
dczmNzaMSt5wmZ+qIlu4nxRiMOaWh5LLntncQoxuAs+sCtZ9bK2c19Urg5WJ615R
d6OWtKINyuVosvlGzquht+ZnejJAgr1XsgF9cCxZonecwYQRlBvOjMRidCTpjzCu
6SEEg/JyiauHq6wZjbz20fXkdD+P8PIV1ZnyUIakDgI7kY0AQHdKh4PSMvDoFpIw
TXU5YrNA8ao1B6CFdyjmLzoY2C9d9SDQTXMX8f8f3GUo9gZ0IzSIFVGFpsKBU0QM
hBgHM6A0WJC9MO3aAKRBcp48y6DXNA==
-----END PRIVATE KEY-----`)

type receiveMail struct {
	HeloName   string
	Sender     string
	Recipients []string
	Data       []byte
}

func runserver(t *testing.T, server *smtpd.Server, received chan receiveMail) (addr string, closer func()) {
	ln, err := net.Listen("tcp", "127.0.1.10:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	go func() {
		server.Handler = func(peer smtpd.Peer, env smtpd.Envelope) error {
			m := receiveMail{
				HeloName:   peer.HeloName,
				Sender:     env.Sender,
				Recipients: env.Recipients,
				Data:       env.Data,
			}
			received <- m
			return nil
		}
		server.Serve(ln)
	}()

	done := make(chan bool)

	go func() {
		<-done
		ln.Close()
	}()

	return ln.Addr().String(), func() {
		done <- true
	}

}

func runsslserver(t *testing.T, server *smtpd.Server, received chan receiveMail) (addr string, closer func()) {
	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	if err != nil {
		t.Fatalf("Cert load failed: %v", err)
	}

	server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return runserver(t, server, received)
}
