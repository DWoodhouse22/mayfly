package keygen

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

func GenerateKeyPair() (*KeyPair, error) {
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		return nil, fmt.Errorf("generating random bytes: %w", err)
	}

	// Curve25519 clamping required by RFC 7748
	// https://www.rfc-editor.org/info/rfc7748/#section-5
	priv[0] &= 0b11111000  // clear low 3 bits: scalar must be a multiple of 8 (curve cofactor)
	priv[31] &= 0b01111111 // clear highest bit: keep scalar below 2^255
	priv[31] |= 0b01000000 // set bit 254: ensure scalar is large enough to be valid

	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("deriving public key: %w", err)
	}

	return &KeyPair{
		PrivateKey: base64.StdEncoding.EncodeToString(priv),
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
	}, nil
}

func EncodeToHex(key string) (string, error) {
	b64Key, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b64Key), nil
}
