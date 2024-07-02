package listener

type Listener interface {
	RawAddress() string
	Address() string
	Close() error
}
