package p2p

import "net"

// peer is an interface that represents the remote node
type Peer interface {
	net.Conn
	Send([]byte) error
	// RemoteAddr() net.Addr
	// Close() error
}

// Transport i anything that can handes the communication
// between nodes in the networkt this can be of the
// form (TCP, UDP, WebSockets, ...)
type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	// ListenAddr() string
}
