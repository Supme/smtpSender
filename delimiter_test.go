package smtpSender

import (
	"bytes"
	"encoding/base64"
	"testing"
)

var  maxLen = 1000
var  delimiter = "\r\n"
func TestDelimitWriter_Write(t *testing.T) {
	texts := make([]string, maxLen)
	for i := 0; i < maxLen; i++ {
		r := make([]rune, i)
		for n := 0; n<i; n++ {
			r[n] = 'O'
		}
		texts[i] = string(r)
	}

	for i, text := range texts {
		delimitBase64 := &bytes.Buffer{}
		dwr := NewDelimitWriter(delimitBase64, []byte(delimiter), 76)
		b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)
		n, err := b64Enc.Write([]byte(text))
		if err != nil {
			t.Errorf("write %d byte error %s", n, err)
		}
		err = b64Enc.Close()
		if err != nil {
			t.Errorf("close error %s", err)
		}
		dst, _ := base64.StdEncoding.DecodeString(delimitBase64.String())
		if text != string(dst) {
			t.Errorf("text lenght %d after delimit base64 wrong \r\n%s\r\nwant\r\n%s\r\n, has\r\n%s", i, delimitBase64.String(), text, string(dst))
		}
	}
}
