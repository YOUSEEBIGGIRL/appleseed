package appleseed

import (
	"errors"
	"github.com/YOUSEEBIGGIRL/appleseed/codec"
	"io"
	"log"
	"net"
	"sync"
)

type Client struct {
	codec codec.ClientCodec
	//reqMu     sync.Mutex // 似乎没什么用，一把锁足以
	request   codec.RequestHeader
	mu        sync.Mutex       // 保护 pending
	globalSeq uint64           // 为 request 分配 seq
	pending   map[uint64]*Call // 保存所有请求，请求完成后，会进行移除
	closing   bool             // user has called Close
	shutdown  bool             // server has told us to stop
}

func NewClient(conn io.ReadWriteCloser) *Client {
	cc := codec.NewGobClientCodec(conn)
	return NewClientWithCodec(cc)
}

func NewClientWithCodec(codec codec.ClientCodec) *Client {
	cli := &Client{
		codec:   codec,
		pending: make(map[uint64]*Call),
	}
	go cli.recv()
	return cli
}

type Call struct {
	ServiceMethod string
	Args          any
	Reply         any
	Error         error
	Done          chan *Call
}

func (c *Call) done() {
	select {
	case c.Done <- c:
	default:
		// 队列已满，请求被丢弃
		log.Println("rpc: client chan capacity is full, this call will be discard")
	}
}

func (c *Client) send(call *Call) {
	//c.reqMu.Lock()
	//defer c.reqMu.Unlock()

	c.mu.Lock()
	seq := c.globalSeq
	c.globalSeq++
	c.pending[seq] = call
	c.mu.Unlock()

	c.request.Seq = seq
	c.request.ServiceMethod = call.ServiceMethod
	if err := c.codec.WriteRequest(&c.request, call.Args); err != nil {
		c.mu.Lock()
		call := c.pending[seq]
		delete(c.pending, seq)
		c.mu.Unlock()

		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (c *Client) recv() {
	var resp codec.ResponseHeader
	var err error
	for err == nil {
		if err = c.codec.ReadResponseHeader(&resp); err != nil {
			log.Println("read response header error: ", err)
			break
		}
		seq := resp.Seq
		c.mu.Lock()
		// 从 pending 中获取对应（seq 相同）的 call，并移除
		call := c.pending[seq]
		delete(c.pending, seq)
		c.mu.Unlock()

		switch {
		case resp.Error != "":
			// 虽然发生了错误，但是仍然需要将连接中的剩余数据（body）消费掉
			// 如果 gob.Decode() 传入的是 nil，那么 gob 会读取连接中的一个值并
			// 将该值丢弃，比如 conn 中使用 gob 序列化了 a，b 两个对象，此时
			// 第一次 decode(nil)，那么 gob 将从 conn 中读取 a 并将其丢弃，
			// 第二次 decode(&b)，gob 会读取下一个值 b
			if err := c.codec.ReadResponseBody(nil); err != nil {
				call.Error = err
			}
			call.Error = errors.New(resp.Error)
			call.done()
		default:
			if err := c.codec.ReadResponseBody(call.Reply); err != nil {
				call.Error = err
			}
			call.done()
		}
	}
	// 发生了 err
	c.mu.Lock()
	c.shutdown = true
	// 连接中没有数据可读了，这种情况可能是服务端已经下线了
	if err == io.EOF {

	}
	// 通知所有剩余的 call 发生了错误
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
}

func (c *Client) Go(serviceMethod string, arg, reply any, done chan *Call) *Call {
	call := new(Call)
	call.ServiceMethod = serviceMethod
	call.Args = arg
	call.Reply = reply
	if done == nil {
		done = make(chan *Call, 10)
	} else {
		if cap(done) == 0 {
			log.Panic("rpc: done channel is unbuffered")
		}
	}
	call.Done = done
	c.send(call)
	return call
}

func (c *Client) Call(serviceMethod string, arg, reply any) error {
	call := <-c.Go(serviceMethod, arg, reply, make(chan *Call, 1)).Done
	return call.Error
}

func Dial(network, addr string) (*Client, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	cli := NewClient(conn)
	return cli, nil
}
