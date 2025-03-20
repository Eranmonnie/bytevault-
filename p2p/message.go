package p2p

//message represents arbitary data that is sent between nodes over each transport
type RPC struct {
	From    string
	Payload []byte
}
