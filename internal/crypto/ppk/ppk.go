package ppk

type PublicKey interface {
	VerifySignature(signature []byte, hash []byte) bool
	AsBytes() []byte
}

type PrivateKey interface {
	CreateSignature(hash []byte) ([]byte, error)
}

type KeyPair struct {
	PublicKey  PublicKey
	PrivateKey PrivateKey
}

func GetPublicKeyFromBytes(bytes []byte) (PublicKey, error) {
	publicKey, err := getECDSAPublicKeyFromBytes(bytes)

	if err != nil {
		return nil, err
	}

	return publicKey, err
}

func GenerateKeyPair() (*KeyPair, error) {
	return generateECDSAKeyPair()
}
