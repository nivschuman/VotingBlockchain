package networking_mocks

type NonceGeneratorMock struct {
}

func (*NonceGeneratorMock) GenerateNonce() (uint64, error) {
	return 1, nil
}
