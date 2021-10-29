package smtpSender_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Supme/smtpSender"
)

const testfolder = "testdata"

func TestDelimitWriter_Write(t *testing.T) {
	compare := func(r1, r2 io.Reader) error {
		d1, err := ioutil.ReadAll(r1)
		if err != nil {
			return fmt.Errorf("r1: %s", err)
		}
		d2, err := ioutil.ReadAll(r2)
		if err != nil {
			return fmt.Errorf("r2: %s", err)
		}
		if len(d1) > len(d2) {
			return fmt.Errorf("r1 size is bigger than r2 %d > %d", len(d1), len(d2))
		}
		if len(d1) < len(d2) {
			return fmt.Errorf("r1 size is smaller than  %d < %d", len(d1), len(d2))
		}
		for i := range d1 {
			if d1[i] != d2[i] {
				return fmt.Errorf("at %d bytes are not equal %x != %x", i, d1[i], d2[i])
			}
		}
		return nil
	}

	files, err := ioutil.ReadDir(testfolder)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		f, err := os.Open(filepath.Join(testfolder, file.Name()))
		if err != nil {
			t.Fatal(err)
		}

		b64buf := &bytes.Buffer{}
		dwr := smtpSender.NewDelimitWriter(b64buf, []byte{0x0d, 0x0a}, 76)
		b64Enc := base64.NewEncoder(base64.StdEncoding, dwr)
		_, err = io.Copy(b64Enc, f)
		if err != nil {
			t.Fatal(err)
		}
		err = b64Enc.Close()
		if err != nil {
			t.Fatal(err)
		}

		dec := base64.NewDecoder(base64.StdEncoding, b64buf)

		_, err = f.Seek(0, 0)
		if err != nil {
			t.Fatal(err)
		}
		err = compare(f, dec)
		if err != nil {
			t.Fatalf("compare %s: %s", f.Name(), err)
		}
	}
}
