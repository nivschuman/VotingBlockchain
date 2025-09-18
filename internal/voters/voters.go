package voters

import (
	"encoding/hex"
	"encoding/json"
	"os"

	ppk "github.com/nivschuman/VotingBlockchain/internal/crypto/ppk"
)

type VotingResult struct {
	CandidateId uint32
	Votes       int
}

type Voter struct {
	Name                string
	KeyPair             ppk.KeyPair
	GovernmentSignature []byte
}

type voterJSON struct {
	Name                string `json:"name"`
	GovernmentSignature string `json:"government_signature"`
	PrivateKeyHex       string `json:"private_key"`
	PublicKeyHex        string `json:"public_key"`
}

func VotersFromJSON(data []byte) ([]*Voter, error) {
	var votersJsonList []voterJSON
	if err := json.Unmarshal(data, &votersJsonList); err != nil {
		return nil, err
	}

	voters := make([]*Voter, 0, len(votersJsonList))

	for _, vj := range votersJsonList {
		privBytes, err := hex.DecodeString(vj.PrivateKeyHex)
		if err != nil {
			return nil, err
		}

		pubBytes, err := hex.DecodeString(vj.PublicKeyHex)
		if err != nil {
			return nil, err
		}

		privKey, err := ppk.GetPrivateKeyFromBytes(privBytes)
		if err != nil {
			return nil, err
		}

		pubKey, err := ppk.GetPublicKeyFromBytes(pubBytes)
		if err != nil {
			return nil, err
		}

		govSig, err := hex.DecodeString(vj.GovernmentSignature)
		if err != nil {
			return nil, err
		}

		voter := &Voter{
			Name: vj.Name,
			KeyPair: ppk.KeyPair{
				PrivateKey: privKey,
				PublicKey:  pubKey,
			},
			GovernmentSignature: govSig,
		}

		voters = append(voters, voter)
	}

	return voters, nil
}

func VotersFromJSONFile(path string) ([]*Voter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return VotersFromJSON(data)
}
