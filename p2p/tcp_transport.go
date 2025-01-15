package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection
type TCPPeer struct {
	conn net.Conn

	// if we  dial and retrieve a conn => outbound = true
	//  if we accept and retrieve a conn => outbound = false
	outbound bool
}

func NewTCPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddress string
	listener      net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(listenAddr string) *TCPTransport {
	return &TCPTransport{
		listenAddress: listenAddr,
		// peers:         make(map[net.Addr]Peer),
	}

}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.listenAddress)
	if err != nil {
		fmt.Printf("TCP listen error: %s\n", err)
		return (err)
	}
	go t.startAcceptLoop()

	return nil

}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
			continue
		}

		go t.handelConn(conn)
	}

}

func (t *TCPTransport) handelConn(conn net.Conn) {
	peer := NewTCPeer(conn, true)
	fmt.Printf(" new incomming connection conn: %+v, peer: %+v\n", conn, peer)
}
