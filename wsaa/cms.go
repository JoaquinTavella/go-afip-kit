package wsaa

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	"go.mozilla.org/pkcs7"
)

// signCMS signs the given data using PKCS#7/CMS (detached=false).
// This produces the signed CMS blob that WSAA's LoginCms expects.
func signCMS(data []byte, cert *x509.Certificate, key *rsa.PrivateKey) ([]byte, error) {
	signedData, err := pkcs7.NewSignedData(data)
	if err != nil {
		return nil, fmt.Errorf("pkcs7 new signed data: %w", err)
	}

	if err := signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{}); err != nil {
		return nil, fmt.Errorf("pkcs7 add signer: %w", err)
	}

	cms, err := signedData.Finish()
	if err != nil {
		return nil, fmt.Errorf("pkcs7 finish: %w", err)
	}

	return cms, nil
}
