package p2p

import (
	"encoding/gob"

	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
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
	// we aer justsetting stream to true so we can handel that in logic
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
