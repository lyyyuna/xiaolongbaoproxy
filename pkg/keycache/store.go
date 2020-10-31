package keycache

import bolt "go.etcd.io/bbolt"

const (
	CABUCKET      = "MITMCA"
	CERTBUCKET    = "MITMCERT"
	CERTKEYBUCKET = "MITMCERTKEY"
)

type CertCache struct {
	Db *bolt.DB
}

func NewCertCache(cachepath string) (*CertCache, error) {
	db, err := bolt.Open(cachepath, 0666, nil)
	if err != nil {
		return nil, err
	}

	certCache := &CertCache{
		Db: db,
	}

	return certCache, nil
}
