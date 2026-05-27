package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type KeyPair struct {
	Private *rsa.PrivateKey
	KeyID   string
}

func LoadOrGenerate(path string) (*KeyPair, error) {
	if data, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse private key: %w", err)
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("private key is not RSA")
			}
			return &KeyPair{Private: rsaKey, KeyID: "key-1"}, nil
		}
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("create key file: %w", err)
	}
	defer f.Close()

	if err := pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: der}); err != nil {
		return nil, fmt.Errorf("write key file: %w", err)
	}

	return &KeyPair{Private: key, KeyID: "key-1"}, nil
}
