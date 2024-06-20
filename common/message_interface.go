package common

type Writer interface {
	WriteMsg(m Encoder) (int, error)
}

type Encoder interface {
	Encode() []byte
}

type Decoder interface {
	Decode(data []byte) error
}
