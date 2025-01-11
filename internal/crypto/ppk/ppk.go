package ppk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"
)

// verify that signature of hash from public key is valid
func VerifySignature(publicKeyCompressed []byte, signature []byte, hash []byte) bool {
	//get public key
	publicKey := UnmarshalCompressedPublicKey(publicKeyCompressed)

	if publicKey == nil {
		return false
	}

	// Unmarshal the signature into r, s components
	var r, s big.Int
	signatureLen := len(signature)
	r.SetBytes(signature[:signatureLen/2])
	s.SetBytes(signature[signatureLen/2:])

	// Use the Verify method from ecdsa to validate the signature
	valid := ecdsa.Verify(publicKey, hash[:], &r, &s)
	return valid
}

func UnmarshalCompressedPublicKey(compressed []byte) *ecdsa.PublicKey {
	//make sure that len of compressed is 33 bytes
	if len(compressed) != 33 {
		return nil
	}

	// The first byte is the prefix (either 0x02 or 0x03)
	prefix := compressed[0]

	// The rest are the x-coordinate bytes (32 bytes)
	x := new(big.Int).SetBytes(compressed[1:])

	// The curve we're using (P-256)
	curve := elliptic.P256()

	// Compute the y-coordinate based on the x-coordinate and prefix
	y := new(big.Int)

	// Calculate the right-hand side of the elliptic curve equation (y² = x³ + ax + b)
	// P-256 curve parameters
	a := new(big.Int)
	b := new(big.Int)
	a.SetInt64(0) // P-256 curve: a = 0
	b.SetInt64(7) // P-256 curve: b = 7
	ySquared := new(big.Int)
	ySquared.Exp(x, big.NewInt(3), nil)
	ySquared.Add(ySquared, a)
	ySquared.Add(ySquared, b)
	ySquared.Mod(ySquared, curve.Params().P) // Mod by the curve's prime

	// Solve for y (y² = ?)
	y.ModSqrt(ySquared, curve.Params().P)

	// If the prefix is 0x03, y should be the odd number
	if prefix == 0x03 && y.Bit(0) == 0 {
		y.Sub(curve.Params().P, y)
	} else if prefix == 0x02 && y.Bit(0) == 1 {
		y.Sub(curve.Params().P, y)
	}

	// Return the reconstructed public key
	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	return pubKey
}

func GenerateKeyPair() (*ecdsa.PublicKey, *ecdsa.PrivateKey, error) {
	curve := elliptic.P256()

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	return &privateKey.PublicKey, privateKey, nil
}

func CreateSignature(privateKey *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])

	if err != nil {
		return nil, err
	}

	return append(r.Bytes(), s.Bytes()...), nil
}

func CompressPublicKey(publicKey *ecdsa.PublicKey) ([]byte, error) {
	// Ensure the elliptic curve is supported (e.g., P-256)
	curve := elliptic.P256()

	// Get the X and Y coordinates of the public key
	x, y := publicKey.X, publicKey.Y

	// Check that the point is on the curve
	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("public key is not on the curve")
	}

	// Compress the public key to 33 bytes
	compressed := make([]byte, 33)
	if y.Bit(0) == 0 {
		compressed[0] = 0x02 // Even Y
	} else {
		compressed[0] = 0x03 // Odd Y
	}

	// Copy the X coordinate (32 bytes)
	xBytes := x.Bytes()
	copy(compressed[1:], xBytes)

	return compressed, nil
}
