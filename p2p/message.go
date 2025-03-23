package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// message represents arbitary data that is sent between nodes over each transport
type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
