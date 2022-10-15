package f860

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"
)

//go:embed encryptionKey.pem
var defaultEncryptionKeyBytes []byte

func DefaultEncryptionKeyBytes() []byte {
	return defaultEncryptionKeyBytes
}

type EncryptionKey struct {
	pub *rsa.PublicKey
}

func ParseEncryptionKey(pubKey []byte) (*EncryptionKey, error) {
	block, _ := pem.Decode(pubKey)
	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("expected 'PUBLIC KEY' block, but got: %s", block.Type)
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse pub key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unsupported pub key type: %T", pub)
	}

	return &EncryptionKey{
		pub: rsaPub,
	}, nil
}

func (k *EncryptionKey) Encrypt(msg []byte) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, k.pub, msg)
}
