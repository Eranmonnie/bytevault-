package p2p

import "net"

// peer is an interface that represents the remote node
type Peer interface {
	net.Conn
	Send([]byte) error
	// RemoteAddr() net.Addr
	CloseStream()
}

// Transport i anything that can handes the communication
// between nodes in the networkt this can be of the
// form (TCP, UDP, WebSockets, ...)
type Transport interface {
	Addr() string
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	// ListenAddr() string
}
