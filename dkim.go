package smtpSender

import (
	"github.com/toorop/go-dkim"
)

func dkimSign(d builderDKIM, body *[]byte) error {
	options := dkim.NewSigOptions()
	options.PrivateKey = d.privateKey
	options.Domain = d.domain
	options.Selector = d.selector
	options.SignatureExpireIn = 3600
	options.BodyLength = 50
	options.Algo = "rsa-sha1"
	options.Headers = []string{"from", "to", "subject"}
	options.AddSignatureTimestamp = true
	options.Canonicalization = "simple/simple"
	return dkim.Sign(body, options)
}
