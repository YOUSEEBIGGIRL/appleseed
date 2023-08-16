package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"

	"github.com/autsu/rpcz/codec"
	"github.com/autsu/rpcz/loadbalance"
	"github.com/autsu/rpcz/registry"
	"github.com/autsu/rpcz/util"
)

var ErrShutdown = errors.New("connection is shut down")

func GetServerAddr(ctx context.Context, reg registry.Client, lb loadbalance.Interface, serviceName string) (string, error) {
	// 从注册中心中获取 serviceName 的所有地址
	addrs, err := reg.Get(ctx, serviceName)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("this service[%v] no address", serviceName)
	}
	for _, a := range addrs {
		lb.Add(loadbalance.Addr{Addr: a})
	}
	// 通过负载均衡选择其中的一个
	return lb.Get(), nil
}

type Client struct {
	//reqMu     sync.Mutex // 似乎没什么用，一把锁足以
	codec         codec.ClientCodec
	requestHeader codec.RequestHeader
	mu            sync.Mutex       // 保护 pending
	globalSeq     uint64           // 为 requestHeader 分配 seq
	pending       map[uint64]*Call // 保存所有请求，请求完成后，会进行移除
	serverAddr    string           // 当前调用的服务的地址，如果 watch 到该地址下线或者变更，可以进行相应的处理
	closing       bool             // 该成员为 true 时，代表客户端主动调用了 Close()
	shutdown      bool             // 该成员为 true 时，代表服务端那边退出了
}

func NewClient(conn io.ReadWriteCloser, serverAddr string) *Client {
	cc := codec.NewGobClientCodec(conn)
	c := newClientWithCodec(cc)
	c.serverAddr = serverAddr
	return c
}

func newClientWithCodec(codec codec.ClientCodec) *Client {
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
		util.Log.Warn("rpc: client chan capacity is full, this call will be discard")
	}
}

func (c *Client) send(call *Call) {
	//c.reqMu.Lock()
	//defer c.reqMu.Unlock()

	// 客户端已经与服务端建立连接，然后在客户端执行 send 前 kill 掉服务端，下面这行会阻塞整个客户端程序（已解决，recv 那边忘写解锁了）
	c.mu.Lock()

	// 如果服务端已经关闭了，那么就没必要往下走了，直接 return 掉，同时调用 call.done()，让 Call() 那边能够消费到，解除阻塞
	if c.shutdown || c.closing {
		c.mu.Unlock()
		call.Error = ErrShutdown
		call.done()
		return
	}

	seq := c.globalSeq
	c.globalSeq++
	c.pending[seq] = call
	c.mu.Unlock()

	c.requestHeader.Seq = seq
	c.requestHeader.ServiceMethod = call.ServiceMethod
	if err := c.codec.WriteRequest(&c.requestHeader, call.Args); err != nil {
		util.Log.Debug("client.send writeRequest error", slog.Any("err", err))
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
		// 保护 pending，因为 send() 那边也会操作 pending
		c.mu.Lock()
		// 从 pending 中获取对应（seq 相同）的 call，并移除
		call := c.pending[seq]
		delete(c.pending, seq)
		c.mu.Unlock()

		switch {
		// 源码里对这一情况也进行了判断，但是注释用机翻完全看不懂，seq 既然是从 response
		// 中获取的，那么怎么可能在 pending 中找不到呢？
		//
		// 想了一下，可能是这个原因：send 中是先写入 seq 到 pending 中，再通过网络发送请求，
		// 但是网络发送可能会出现错误，此时便会将这个 seq 从 pending 中移除，就可能出
		// 现 pending 中不存在这个 seq，所以值就为 nil。
		//
		// 至于为什么 recv 这边能读到 seq，可能是因为 WriteRequest 是先写入请求头，再写入
		// 请求体，所以可能请求头写入成功（里面有 seq），但是请求体写入失败，此时请求体已经发送
		// 给对方了，但是 WriteRequest 因为请求体写入失败而返回 err，导致这个 seq 从 pending
		// 中删除，就会出现能读到 seq，但是无法从 pending 中找到的情况。（这里还是有点存疑，
		// 因为 WriteRequest 最后还会调用 c.encBuf.Flush()，不确定是 c.enc.Encode 时就会
		// 立马写入，还是等到最后调用 Flush 时才写入）
		//
		// 疑问：那可不可以让网络请求在前，写入 pending 在后呢？
		// 这样可能依然会有问题，因为 recv 这边是先读网络请求，获得 seq，再从 pending 中查找，
		// 如果 send 这边是网络请求在前， 写入 pending 在后，那么可能存在 recv 的 pending
		// 查找操作发生在 send 写入 pending 之前，此时 pending 还没有写入这个 seq，所以依然
		// 会找不到，导致 call == nil
		case call == nil:
			err = c.codec.ReadResponseBody(nil)
			if err != nil {
				err = errors.New("reading error body: " + err.Error())
			}
		case resp.Error != "":
			call.Error = errors.New(resp.Error)
			// 虽然发生了错误，但是仍然需要将连接中的剩余数据（body）消费掉
			// 如果 gob.Decode() 传入的是 nil，那么 gob 会读取连接中的一个值并
			// 将该值丢弃，比如 conn 中使用 gob 序列化了 a，b 两个对象，此时
			// 第一次 decode(nil)，那么 gob 将从 conn 中读取 a 并将其丢弃，
			// 第二次 decode(&b)，gob 会读取下一个值 b
			if err := c.codec.ReadResponseBody(nil); err != nil {
				call.Error = err
			}
			call.done()
		default:
			if err := c.codec.ReadResponseBody(call.Reply); err != nil {
				call.Error = err
			}
			call.done()
		}
	}
	// 如果流程走到这里，说明发生了 err
	// 这个锁保护 closing 以及 pending
	c.mu.Lock()
	c.shutdown = true
	// 连接中没有数据可读了，这种情况可能是服务端已经下线了
	if err == io.EOF {
		if c.closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	// 通知所有剩余的 call 发生了错误
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
	c.mu.Unlock()
}

func (c *Client) Go(ctx context.Context, serviceMethod string, arg, reply any, done chan *Call) *Call {
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

	select {
	case <-ctx.Done():
		util.Log.Debug("client.Go: time out")
		call.Error = errors.New("rpc call error: time out")
		call.done()
		return call
	default:
	}

	c.send(call)
	return call
}

func (c *Client) Call(ctx context.Context, serviceMethod string, arg, reply any) error {
	call := <-c.Go(ctx, serviceMethod, arg, reply, make(chan *Call, 1)).Done
	return call.Error
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		return ErrShutdown
	}
	c.closing = true
	c.mu.Unlock()
	return c.codec.Close()
}
