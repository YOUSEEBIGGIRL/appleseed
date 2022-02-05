package easyrpc

import (
	"easyrpc/codec"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// 发生错误时，将该空结构体作为 body 发送
var invalidRequest = struct{}{}

type Server struct {
	service
	mu       sync.Mutex
	wg       sync.WaitGroup
	reqPool  *sync.Pool
	respPool *sync.Pool
}

func NewServer() *Server {
	return &Server{
		reqPool: &sync.Pool{
			New: func() interface{} {
				return &codec.RequestHeader{}
			},
		},
		respPool: &sync.Pool{
			New: func() interface{} {
				return &codec.ResponseHeader{}
			},
		},
	}
}

func (s *Server) RunWithTCP(host, port string) error {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return err
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go s.serverConn(conn)
	}
}

func (s *Server) serverConn(conn net.Conn) {
	//var head Header
	//if err := gob.NewDecoder(conn).Decode(&head); err != nil {
	//	log.Println("rpc server: decode header error: ", err)
	//	return
	//}
	//log.Printf("%+v \n", head)
	//
	//var c Codec
	//switch head.CodecType {
	//case Gob:
	//	c = NewGobCodec(conn)
	//default:
	//	log.Printf("rpc server: invalid codec type: %v\n", head.CodecType)
	//	return
	//}

	c := codec.NewGobCodec(conn)
	s.ServerCodec(c)
}

func (s *Server) ServerCodec(c codec.Codec) {
	for {
		// 读取 request
		req, err := s.readRequest(c)
		if err != nil {
			if req == nil {
				break
			}
			req.head.Error = err.Error()
			if err := s.sendResponse(c, req.head, invalidRequest); err != nil {
				log.Println(err)
			}
			continue
		}

		s.wg.Add(1)
		// 处理请求，处理完成后，该函数内部会发送响应数据
		go s.handlerRequest(c, req)
	}
	s.wg.Wait()
	c.Close()
}

func (s *Server) readRequestHeader(c codec.Codec) (*codec.RequestHeader, error) {
	h := s.reqPool.Get().(*codec.RequestHeader)
	if err := c.ReadRequestHeader(h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rcp server: read header error: ", err)
		}
		return nil, err
	}
	log.Printf("%+v \n", h)
	return h, nil
}

func (s *Server) readRequest(c codec.Codec) (*codec.RequestHeader, error) {
	header, err := s.readRequestHeader(c)
	if err != nil {
		return nil, err
	}

	req := &request{head: header}

	if err := c.ReadBody(req.param); err != nil {
		log.Println("rpc server: read argv err: ", err)
	}
	return req, nil
}

func (s *Server) sendResponse(c Codec, head *Header, body interface{}) error {
	// 加锁保证回复的消息与请求是一一对应的，不会存在乱序问题
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := c.Write(head, body); err != nil {
		log.Println("rpc server: write response err: ", err)
		return err
	}
	return nil
}

func (s *Server) handlerRequest(c codec.Codec, req *codec.RequestHeader) {
	defer s.wg.Done()
	log.Println(req.head, req.result)

	if err := s.service.call(req.ServiceMethod, req.param, req.result); err != nil {
		log.Println(err)
	}
	log.Printf("%+v \n", req.param)
	log.Printf("%+v \n", req.result)
	//req.replyv = reflect.ValueOf(fmt.Sprintf("rpc resp %d", req.head.Seq))
	if err := s.sendResponse(c, req.head, req.result); err != nil {
		log.Println(err)
	}
}
