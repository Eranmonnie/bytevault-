package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWSTransport(t *testing.T) {
	listenAddr := ":3000"
	WSOpts := WebSocketTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: NOPHandshakeFunc,
		Decoder:       WSDecoder{},
	}
	tr := NewWebSocketTransport(WSOpts)
	assert.Equal(t, tr.ListenAddr, listenAddr)

	assert.Nil(t, tr.ListenAndAccept())

}
