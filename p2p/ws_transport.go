package p2p

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketPeer represents a remote node connected via WebSocket
type WebSocketPeer struct {
	conn     *websocket.Conn
	outbound bool
	wg       *sync.WaitGroup
	sendMu   sync.Mutex // Mutex for concurrent writes to WebSocket
}

func NewWebSocketPeer(conn *websocket.Conn, outbound bool) *WebSocketPeer {
	return &WebSocketPeer{
		conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

// Implement net.Conn methods (required for Peer interface)
func (p *WebSocketPeer) Read(b []byte) (n int, err error) {
	// WebSockets use a message-based model, so we need to adapt to io.Reader model
	_, message, err := p.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	return copy(b, message), nil
}

func (p *WebSocketPeer) Write(b []byte) (n int, err error) {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()
	err = p.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (p *WebSocketPeer) Send(b []byte) error {
	_, err := p.Write(b)
	return err
}

func (p *WebSocketPeer) Close() error {
	return p.conn.Close()
}

func (p *WebSocketPeer) CloseStream() {
	p.wg.Done()
}

// Other required net.Conn methods
func (p *WebSocketPeer) LocalAddr() net.Addr  { return p.conn.LocalAddr() }
func (p *WebSocketPeer) RemoteAddr() net.Addr { return p.conn.RemoteAddr() }
func (p *WebSocketPeer) SetDeadline(t time.Time) error {
	if err := p.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return p.conn.SetWriteDeadline(t)
}
func (p *WebSocketPeer) SetReadDeadline(t time.Time) error  { return p.conn.SetReadDeadline(t) }
func (p *WebSocketPeer) SetWriteDeadline(t time.Time) error { return p.conn.SetWriteDeadline(t) }

// WebSocketTransportOpts contains options for the WebSocket transport
type WebSocketTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       DecoderWS
	OnPeer        func(Peer) error
	EnableTLS     bool
	CertFile      string
	KeyFile       string
}

// WebSocketTransport implements the Transport interface using WebSockets
type WebSocketTransport struct {
	WebSocketTransportOpts
	upgrader websocket.Upgrader
	server   *http.Server
	rpcch    chan RPC
	peers    map[string]*WebSocketPeer
	peersMu  sync.Mutex
	dialer   *websocket.Dialer
}

func NewWebSocketTransport(opts WebSocketTransportOpts) *WebSocketTransport {
	return &WebSocketTransport{
		WebSocketTransportOpts: opts,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all connections
			},
		},
		rpcch:   make(chan RPC, 1024),
		peers:   make(map[string]*WebSocketPeer),
		peersMu: sync.Mutex{},
		dialer:  websocket.DefaultDialer,
	}
}

func (t *WebSocketTransport) Consume() <-chan RPC {

	return t.rpcch
}

func (t *WebSocketTransport) Addr() string {
	return t.ListenAddr
}

func (t *WebSocketTransport) Close() error {
	if t.server != nil {
		return t.server.Close()
	}
	return nil
}

func (t *WebSocketTransport) Dial(addr string) error {
	url := fmt.Sprintf("ws://%s/ws", addr)
	if t.EnableTLS {
		url = fmt.Sprintf("wss://%s/ws", addr)
	}

	conn, _, err := t.dialer.Dial(url, nil)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)
	return nil
}

func (t *WebSocketTransport) ListenAndAccept() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", t.wsHandler)

	t.server = &http.Server{
		Addr:    t.ListenAddr,
		Handler: mux,
	}

	log.Printf("WebSocket transport listening on %s", t.ListenAddr)

	go func() {
		var err error
		if t.EnableTLS {
			err = t.server.ListenAndServeTLS(t.CertFile, t.KeyFile)
		} else {
			err = t.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	return nil
}

func (t *WebSocketTransport) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	t.handleConn(conn, false)
}

func (t *WebSocketTransport) handleConn(conn *websocket.Conn, outbound bool) {
	var err error
	defer func() {
		t.peersMu.Lock()
		delete(t.peers, conn.RemoteAddr().String())
		t.peersMu.Unlock()
		if err != nil {
			log.Printf("WebSocket connection closed with error: %v", err)

		}
		conn.Close()
	}()

	peer := NewWebSocketPeer(conn, outbound)

	// Perform handshake
	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	// Register with OnPeer callback
	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Add to peers map
	t.peersMu.Lock()
	t.peers[conn.RemoteAddr().String()] = peer
	t.peersMu.Unlock()

	// Start read loop
	for {
		rpc := RPC{}
		err := t.Decoder.Decode(conn, &rpc)

		if err != nil {
			return
		}

		rpc.From = conn.RemoteAddr().String()
		if rpc.Stream {
			peer.wg.Add(1)
			fmt.Printf("incoming stream from %s waiting... \n", rpc.From)
			peer.wg.Wait()
			fmt.Printf("stream is done, continuing normal read loop\n")
			continue
		}
		t.rpcch <- rpc

	}
}
