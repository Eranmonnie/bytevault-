package p2p

// peer is an interface that represents the remote node
type Peer interface {
	Close() error
}

// Transport anything that can handes the communication
// between nodes in the networkt this can be of the
// form (TCP, UDP, WebSockets, ...)
type Transport interface {
	listenAndAccept() error
	COnsume() <-chan RPC
}
