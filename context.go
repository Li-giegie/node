package node

type nodeContext struct {
	*message
	write       func(m *message) error
	setRespChan func(key any) (value any, ok bool)
}
