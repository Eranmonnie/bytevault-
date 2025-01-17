package p2p

import "net"

//message represents arbitary data that is sent between nodes over each transport
type Message struct {
	From    net.Addr
	Payload []byte
}
