package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"time"
)

// generateKey generates a new RSA keypair if none exists and writes it to the disk in PEM format.
func generateKey() error {
	if _, err := os.Stat("files/cert.pem"); !errors.Is(err, os.ErrNotExist) {
		return nil // cert already exists
	}
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Meldeplattform"},
			Country:       []string{"DE"},
			Province:      []string{"BY"},
			Locality:      []string{"Munich"},
			StreetAddress: []string{"Arcisstraße 21"},
			PostalCode:    []string{"80333"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(2, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}
	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return err
	}
	file, err := os.OpenFile("files/cert.pem", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(caPEM.Bytes())

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return err
	}
	keyF, err := os.OpenFile("files/key.pem", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer keyF.Close()
	_, err = keyF.Write(caPrivKeyPEM.Bytes())
	return nil
}
