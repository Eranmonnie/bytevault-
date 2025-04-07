package p2p

import (
	"encoding/gob"
	"fmt"

	"io"

	"github.com/gorilla/websocket"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type DecoderWS interface {
	Decode(conn *websocket.Conn, msg *RPC) error
}

type GOBDecder struct{}

func (dec GOBDecder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {

	peakBuf := make([]byte, 1)
	if _, err := r.Read(peakBuf); err != nil {
		return nil
	}

	// incase of a stream we are decoding what is being sent over the network
	// we are just setting stream to true so we can handel that in logic
	stream := peakBuf[0] == IncomingStream
	if stream {
		msg.Stream = true
		return nil
	}

	buff := make([]byte, 1028)
	n, err := r.Read(buff)
	if err != nil {
		return err
	}
	msg.Payload = buff[:n]

	return nil

}

type WSDecoder struct{}

func (dec WSDecoder) Decode(conn *websocket.Conn, msg *RPC) error {
	// Read the message type and payload from the WebSocket connection
	messageType, p, err := conn.ReadMessage()
	if err != nil {
		return err
	}

	if messageType != websocket.BinaryMessage {
		return fmt.Errorf("unsupported message type: %d", messageType)
	}

	if len(p) == 0 {
		return fmt.Errorf("received empty message")
	}

	switch p[0] {
	case IncomingStream:
		msg.Stream = true
		msg.Payload = nil
		return nil

	case IncomingMessage:

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read message payload: %w", err)
		}

		if messageType != websocket.BinaryMessage {
			return fmt.Errorf("unsupported payload message type: %d", messageType)
		}

		msg.Payload = payload
		msg.Stream = false
		return nil

	default:
		return fmt.Errorf("unknown message prefix: %d", p[0])
	}
}
