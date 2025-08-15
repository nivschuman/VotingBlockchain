package ppk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
)

type ECDSAPublicKey struct {
	publicKey *ecdsa.PublicKey
}

type ECDSAPrivateKey struct {
	privateKey *ecdsa.PrivateKey
}

func (ecdsaPublicKey *ECDSAPublicKey) VerifySignature(signature []byte, hash []byte) bool {
	// Use the Verify method from ecdsa to validate the signature
	valid := ecdsa.VerifyASN1(ecdsaPublicKey.publicKey, hash, signature)
	return valid
}

func (ecdsaPublicKey *ECDSAPublicKey) AsBytes() []byte {
	publicKey := ecdsaPublicKey.publicKey
	return elliptic.MarshalCompressed(publicKey.Curve, publicKey.X, publicKey.Y)
}

func (ecdsaPrivateKey *ECDSAPrivateKey) AsBytes() ([]byte, error) {
	privateKey := ecdsaPrivateKey.privateKey
	return x509.MarshalECPrivateKey(privateKey)
}

func (ecdsaPrivateKey *ECDSAPrivateKey) CreateSignature(hash []byte) ([]byte, error) {
	signature, err := ecdsa.SignASN1(rand.Reader, ecdsaPrivateKey.privateKey, hash)

	if err != nil {
		return nil, err
	}

	return signature, nil
}

func getECDSAPublicKeyFromBytes(bytes []byte) (*ECDSAPublicKey, error) {
	// Ensure the compressed key is valid
	if len(bytes) != 33 {
		return nil, fmt.Errorf("invalid compressed key length: %d", len(bytes))
	}

	// Use P-256 curve
	curve := elliptic.P256()

	// UnmarshalCompressed reconstructs the public key
	x, y := elliptic.UnmarshalCompressed(curve, bytes)
	if x == nil || y == nil {
		return nil, fmt.Errorf("invalid compressed public key")
	}

	// Return the reconstructed public key
	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	publicKey := &ECDSAPublicKey{
		publicKey: pubKey,
	}

	return publicKey, nil
}

func getECDSAPrivateKeyFromBytes(bytes []byte) (*ECDSAPrivateKey, error) {
	privKey, err := x509.ParseECPrivateKey(bytes)
	if err != nil {
		return nil, err
	}

	privateKey := &ECDSAPrivateKey{
		privateKey: privKey,
	}

	return privateKey, nil
}

func generateECDSAKeyPair() (*KeyPair, error) {
	curve := elliptic.P256()

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	keyPair := &KeyPair{
		PublicKey: &ECDSAPublicKey{
			publicKey: &privateKey.PublicKey,
		},
		PrivateKey: &ECDSAPrivateKey{
			privateKey: privateKey,
		},
	}

	return keyPair, nil
}
