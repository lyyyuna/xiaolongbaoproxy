package key

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"time"
)

const (
	PEM_HEADER_PRIVATE_KEY = "RSA PRIVATE KEY"
	PEM_HEADER_PUBLIC_KEY  = "RSA PUBLIC KEY"
	PEM_HEADER_CERTIFICATE = "CERTIFICATE"
)

// PrivateKey is a convenience wrapper for rsa.PrivateKey
type PrivateKey struct {
	rsaKey *rsa.PrivateKey
}

// Certificate is a convenience wrapper for x509.Certificate
type Certificate struct {
	cert     *x509.Certificate
	derBytes []byte
}

// LoadPKFromFile loads private key from the specified file
func LoadPKFromFile(filename string) (*PrivateKey, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("unable to decode the pem file")
	}
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to decode x509 private key")
	}

	return &PrivateKey{rsaKey: rsaKey}, nil
}

// LoadCertificateFromFile loads certificate from the specified file
func LoadCertificateFromFile(filename string) (*Certificate, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("unable to decode the pem file")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to decode x509 certificate")
	}

	return &Certificate{cert: cert}, nil
}

func CertificateForKey(CN string, key *PrivateKey, ca *Certificate) (*Certificate, error) {
	// set up our server certificate template
	template := &x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(int64(time.Now().UnixNano())),
		Subject: pkix.Name{
			Organization: []string{"MITM, INC."},
			CommonName:   CN,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	// If name is an ip address, add it as an IP SAN
	ip := net.ParseIP(CN)
	if ip != nil {
		template.IPAddresses = []net.IP{ip}
	}

	// sign the cert with CA
	signedBytes, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.rsaKey.PublicKey, key.rsaKey)
	if err != nil {
		return nil, err
	}

	signedCert, err := x509.ParseCertificate(signedBytes)
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: signedCert, derBytes: signedBytes}, nil
}

func (k *PrivateKey) pemBlock() *pem.Block {
	return &pem.Block{Type: PEM_HEADER_PRIVATE_KEY, Bytes: x509.MarshalPKCS1PrivateKey(k.rsaKey)}
}

func (k *PrivateKey) PEMEncoded() (pemBytes []byte) {
	return pem.EncodeToMemory(k.pemBlock())
}

func (c *Certificate) pemBlock() *pem.Block {
	return &pem.Block{Type: PEM_HEADER_CERTIFICATE, Bytes: c.derBytes}
}

func (c *Certificate) PEMEncoded() (pemBytes []byte) {
	return pem.EncodeToMemory(c.pemBlock())
}
