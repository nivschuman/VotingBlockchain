package networking_models

type GetData struct {
	inv *Inv
}

func NewGetData() *GetData {
	return &GetData{
		NewInv(),
	}
}

func NewGetDataMessage(getData *GetData) (*Message, error) {
	getDataBytes, err := getData.AsBytes()

	if err != nil {
		return nil, err
	}

	return NewMessage(CommandGetData, getDataBytes), nil
}

func (getData *GetData) Items() []InvItem {
	return getData.inv.Items
}

func (getData *GetData) AddItem(itemType uint32, itemHash []byte) {
	getData.inv.AddItem(itemType, itemHash)
}

func (getData *GetData) AsBytes() ([]byte, error) {
	return getData.inv.AsBytes()
}

func GetDataFromBytes(data []byte) (*GetData, error) {
	inv, err := InvFromBytes(data)

	if err != nil {
		return nil, err
	}

	return &GetData{inv: inv}, nil
}
