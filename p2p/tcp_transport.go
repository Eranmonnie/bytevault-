package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection
type TCPPeer struct {
	net.Conn

	// if we  dial and retrieve a conn => outbound = true
	//  if we accept and retrieve a conn => outbound = false
	outbound bool
	WG       *sync.WaitGroup
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

func NewTCPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		WG:       &sync.WaitGroup{},
	}
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}

}

// func (t *TCPTransport) ListenAddr() string{
//   return t.ListenAddr
// }

// consume imlements the transport interface
// which will return a read only channel recieved from another peer in the network
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// Close implements the transport interface
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// dial implements the transport interface
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil
	}
	go t.handelConn(conn, true)
	return nil
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		fmt.Printf("TCP listen error: %s\n", err)
		return (err)
	}
	go t.startAcceptLoop()

	log.Printf("TCP transport listening on port: %s\n", t.ListenAddr)

	return nil

}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
			// continue
		}

		go t.handelConn(conn, false)
	}

}

type Temp struct{}

func (t *TCPTransport) handelConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {

		fmt.Printf("TCP close error dropping: %s\n", err)
		conn.Close()
	}()
	peer := NewTCPeer(conn, outbound)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// read loop
	rpc := RPC{}
	for {
		err := t.Decoder.Decode(conn, &rpc)

		if err != nil {
			return
		}
		rpc.From = conn.RemoteAddr().String()
		peer.WG.Add(1)
		fmt.Println("waiting till stream is done")
		t.rpcch <- rpc
		peer.WG.Wait()
		fmt.Println("stream is done, continuing normal read loop ")
	}
}
