package compress

type Nop struct {
}

func NewNop() Nop {
	return Nop{}
}

func (n Nop) Encode(data []byte) ([]byte, error) {
	return data, nil
}

func (n Nop) Decode(data []byte) ([]byte, error) {
	return data, nil
}
