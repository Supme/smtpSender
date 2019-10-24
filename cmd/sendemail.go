// Example use:
// sendemail -V -f from@domain.tld -t to@domain.tld -s "Hello subject!" -m "Hello, world!"
// sendemail -f from@domain.tld -t to@domain.tld -s "Hello subject!" -html ./message.html -amp ./amp.html -txt ./message.txt
package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/Supme/smtpSender"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	fromEmail        string
	fromName         string
	toEmail          string
	toName           string
	subject          string
	textPartFile     string
	ampPartFile      string
	htmlPartFile     string
	htmlRelatedFiles string
	attachmentFiles  string
	message          string
	hostname         string
	smtpServer       string
	smtpUser         string
	smtpPassword     string
	dkimDomain       string
	dkimSelector     string
	dkimKeyFile      string
)

type buffer struct {
	bytes.Buffer
}

func (b *buffer) Close() error {
	return nil
}

func main() {
	flag.StringVar(&fromEmail, "f", "", "From email")
	flag.StringVar(&fromName, "fn", "", "From name")
	flag.StringVar(&toEmail, "t", "", "To email")
	flag.StringVar(&toName, "tn", "", "To name")
	flag.StringVar(&message, "m", "", "Text email message")
	flag.StringVar(&subject, "s", "", "Email subject")
	flag.StringVar(&textPartFile, "txt", "", "Text part file")
	flag.StringVar(&ampPartFile, "amp", "", "AMP part file")
	flag.StringVar(&htmlPartFile, "html", "", "HTML part file")
	flag.StringVar(&htmlRelatedFiles, "htmlrfs", "", "HTML related file split comma (',') separator")
	flag.StringVar(&attachmentFiles, "att", "", "Attachment files split comma (',') separator")
	flag.StringVar(&hostname, "h", "", "Hostname for direct send (if blank, use resolved IP)")
	flag.StringVar(&smtpServer, "S", "", "SMTP server, if not set, use direct send")
	flag.StringVar(&smtpUser, "u", "", "SMTP user")
	flag.StringVar(&smtpPassword, "p", "", "SMTP password")
	flag.StringVar(&dkimDomain, "dd", "", "DKIM domain")
	flag.StringVar(&dkimSelector, "ds", "", "DKIM selector")
	flag.StringVar(&dkimKeyFile, "df", "", "DKIM private key file")
	verbose := flag.Bool("V", false, "Verbose message")
	notSend := flag.Bool("N", false, "Not send to server")

	version := flag.Bool("v", false, "Prints version")
	flag.Parse()
	if *version {
		fmt.Printf("Sendemail version: v%s\r\n\r\n", smtpSender.Version)
		os.Exit(0)
	}

	if *verbose {
		fmt.Printf(
			"Use parameters:\r\n\tfromEmail: %s\r\n\tfromName: %s\r\n\ttoEmail: %s\r\n\ttoName: %s\r\n\tsubject: %s\r\n\ttextPartFile: %s\r\n\tampPartFile: %s\r\n\thtmlPartFile: %s\r\n\thtmlRelatedFiles: %s\r\n\tattachmentFiles: %s\r\n\tmessage: %s\r\n\thostname: %s\r\n\tsmtpServer: %s\r\n\tsmtpUser: %s\r\n\tsmtpPassword: %s\r\n\tdkimDomain: %s\r\n\tdkimSelector: %s\r\n\tdkimKeyFile: %s\r\n",
			fromEmail, fromName, toEmail, toName, subject, textPartFile, ampPartFile, htmlPartFile, htmlRelatedFiles, attachmentFiles, message, hostname, smtpServer, smtpUser, smtpPassword, dkimDomain, dkimSelector, dkimKeyFile,
		)
	}

	bldr := smtpSender.NewBuilder()
	bldr.SetFrom(fromName, fromEmail).SetTo(toName, toEmail).SetSubject(subject).AddTextPart([]byte(message))
	if textPartFile != "" {
		bldr.AddTextFunc(func(w io.Writer) error {
			f, err := os.Open(textPartFile)
			if err != nil {
				return fmt.Errorf("open text part file: %s", err)
			}
			_, err = io.Copy(w, f)
			if err != nil {
				return fmt.Errorf("send text part file: %s", err)
			}
			return nil
		})
	}

	if ampPartFile != "" {
		bldr.AddAMPFunc(func(w io.Writer) error {
			f, err := os.Open(ampPartFile)
			if err != nil {
				return fmt.Errorf("open amp part file: %s", err)
			}
			_, err = io.Copy(w, f)
			if err != nil {
				return fmt.Errorf("send amp part file: %s", err)
			}
			return nil
		})
	}

	if htmlPartFile != "" {
		var related []string
		if htmlRelatedFiles != "" {
			related = strings.Split(htmlRelatedFiles, ",")
		}

		err := bldr.AddHTMLFunc(func(w io.Writer) error {
			f, err := os.Open(htmlPartFile)
			if err != nil {
				return fmt.Errorf("open html part file: %s", err)
			}
			_, err = io.Copy(w, f)
			if err != nil {
				return fmt.Errorf("send html part file: %s", err)
			}
			return nil
		}, related...)
		if err != nil {
			log.Fatalf("add html part: %s", err)
		}
	}

	if attachmentFiles != "" {
		attachments := strings.Split(attachmentFiles, ",")
		err := bldr.AddAttachment(attachments...)
		if err != nil {
			log.Fatalf("add attachment files: %s", err)
		}
	}

	if dkimDomain != "" && dkimSelector != "" && dkimKeyFile != "" {
		privateKey, err := ioutil.ReadFile(dkimKeyFile)
		if err != nil {
			log.Fatalf("read DKIM private key file: %s", err)
		}
		bldr.SetDKIM(dkimDomain, dkimSelector, privateKey)
	}

	wg := &sync.WaitGroup{}

	email := bldr.Email("", func(result smtpSender.Result) {
		fmt.Printf("Result for email duration: %f sec result: %v\n", result.Duration.Seconds(), result.Err)
		wg.Done()
	})

	if *verbose {
		buf := &buffer{}
		err := email.WriteCloser(buf)
		if err != nil {
			log.Fatalf("write email to buffer: %s", err)
		}
		fmt.Printf("\r\n--- Message body ---\r\n%s--- End message body ---\r\n\r\n", buf.String())
	}

	if !*notSend {
		fmt.Println("Send and wait result...")
		wg.Add(1)
		conn := new(smtpSender.Connect)
		conn.SetHostName(hostname)
		if smtpServer != "" && smtpUser != "" && smtpPassword != "" {
			host, portStr, err := net.SplitHostPort(smtpServer)
			if err != nil {
				log.Fatalf("split host port SMTP server: %s", err)
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				log.Fatalf("parse SMTP server port: %s", err)
			}
			server := &smtpSender.SMTPserver{
				Host:     host,
				Port:     port,
				Username: smtpUser,
				Password: smtpPassword,
			}
			email.Send(conn, server)
		} else {
			email.Send(conn, nil)
		}

		wg.Wait()
		fmt.Println("Done")
	}
}
