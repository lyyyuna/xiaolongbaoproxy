package keycache

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"
	"xiaolongbaoproxy/pkg/key"

	bolt "go.etcd.io/bbolt"
)

const (
	CABUCKET   = "MITMCA"
	ROOTCACERT = "MITMROOTCACERT"
	ROOTCAKEY  = "MITMROOTCAKEY"
	CERTBUCKET = "MITMCERT"
	KEYBUCKET  = "MITMKEY"
)

type CertCache struct {
	Db *bolt.DB
}

func sliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func NewCertCache(cachepath string, cacert *key.Certificate, cakey *key.PrivateKey) (*CertCache, error) {
	db, err := bolt.Open(cachepath, 0666, nil)
	if err != nil {
		return nil, err
	}

	// reset cache store if ca cert has changed
	err = db.Update(func(t *bolt.Tx) error {
		b, err := t.CreateBucketIfNotExists([]byte(CABUCKET))
		if err != nil {
			return err
		}

		cacheCaCert := b.Get([]byte(ROOTCACERT))
		cacheCaKey := b.Get([]byte(ROOTCAKEY))
		if !sliceEqual(cacheCaCert, cacert.PEMEncoded()) || !sliceEqual(cacheCaKey, cakey.PEMEncoded()) {
			err := b.Put([]byte(ROOTCACERT), []byte(cacert.PEMEncoded()))
			if err != nil {
				return err
			}
			err = b.Put([]byte(ROOTCAKEY), []byte(cakey.PEMEncoded()))
			if err != nil {
				return err
			}

			// remove other buckets as root ca is changed
			t.DeleteBucket([]byte(CERTBUCKET))
			t.DeleteBucket([]byte(KEYBUCKET))
		}

		_, err = t.CreateBucketIfNotExists([]byte(CERTBUCKET))
		if err != nil {
			return err
		}

		_, err = t.CreateBucketIfNotExists([]byte(KEYBUCKET))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	certCache := &CertCache{
		Db: db,
	}

	return certCache, nil
}

func (c *CertCache) GetKeyPair(host string) (*tls.Certificate, error) {
	var certDerBytes []byte
	var keyBytes []byte
	err := c.Db.Update(func(t *bolt.Tx) error {
		certBucket := t.Bucket([]byte(CERTBUCKET))
		certDerBytes = certBucket.Get([]byte(host))

		keyBucket := t.Bucket([]byte(KEYBUCKET))
		keyBytes = keyBucket.Get([]byte(host))

		if certDerBytes == nil || keyBytes == nil {
			return errors.New("cert/key not found")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// check cert `not after` time is valid
	cert, err := x509.ParseCertificate(certDerBytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate error: %v", err)
	}
	// the cert from cache's time is invalid
	if cert.NotAfter.Before(time.Now()) {
		return nil, errors.New("the cert's `not after` is before now")
	}

	wrappedCert := &key.Certificate{Cert: cert, DerBytes: certDerBytes}

	keypair, err := tls.X509KeyPair(wrappedCert.PEMEncoded(), keyBytes)
	if err != nil {
		return nil, err
	}

	return &keypair, err
}

func (c *CertCache) SetKeyPair(host string, certBytes, keyBytes []byte) error {

	err := c.Db.Update(func(t *bolt.Tx) error {
		certBucket := t.Bucket([]byte(CERTBUCKET))
		err := certBucket.Put([]byte(host), certBytes)
		if err != nil {
			return err
		}

		keyBucket := t.Bucket([]byte(KEYBUCKET))
		err = keyBucket.Put([]byte(host), keyBytes)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
