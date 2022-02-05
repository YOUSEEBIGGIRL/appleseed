package easyrpc

import "sync"

type ClientCodec interface {
	WriteRequest(*request, interface{}) error
	ReadResponseHeader(*response) error
	ReadResponseBody(interface{}) error

	Close() error
}

type Client struct {
	codec ClientCodec

	reqMu sync.Mutex // protects following
	request  request

	mu    sync.Mutex // protects following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

// Call represents an active RPC.
type Call struct {
	ServiceMethod string      // The name of the service and method to call.
	Args          interface{} // The argument to the function (*struct).
	Reply         interface{} // The reply from the function (*struct).
	Error         error       // After completion, the error status.
	Done          chan *Call  // Receives *Call when Go is complete.
}

func (c *Client) send(call *Call) {
	c.reqMu.Lock()
	defer c.reqMu.Unlock()

	req := c.request
	req.param =
}


