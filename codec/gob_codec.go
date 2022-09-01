package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobServerCodec struct {
	conn    io.ReadWriteCloser // 通常是通过 TCP 或者 Unix 建立 socket 时得到的连接
	buf     *bufio.Writer      // 包装 socket conn
	decoder *gob.Decoder       // gob 解码
	encoder *gob.Encoder       // gob 编码
	closed  bool               // 防止重复关闭
}

func NewGobServerCodec(conn io.ReadWriteCloser) ServerCodec {
	buf := bufio.NewWriter(conn)
	return &GobServerCodec{
		conn:    conn,
		buf:     buf,
		decoder: gob.NewDecoder(conn), // 从 conn 中读取数据，并用 gob 解析出来
		encoder: gob.NewEncoder(buf),  // 将数据写入到 buf 中，并用 gob 编码数据
	}
}

func (g *GobServerCodec) Close() error {
	if g.closed {
		return nil
	}
	g.closed = true
	return g.conn.Close()
}

// ReadRequestHeader 从 conn 的数据中，使用 gob 解析出 header 部分
func (g *GobServerCodec) ReadRequestHeader(req *RequestHeader) error {
	return g.decoder.Decode(req)
}

// ReadRequestBody 从 conn 的数据中，使用 gob 解析出 body 部分
func (g *GobServerCodec) ReadRequestBody(body any) error {
	return g.decoder.Decode(body)
}

// WriteResponse 使用 gob 对 head 和 body 进行编码，并写入到 conn 中
func (g *GobServerCodec) WriteResponse(resp *ResponseHeader, body any) error {
	defer func() {
		err := g.buf.Flush() // 最后写入数据到 conn
		if err != nil {
			g.conn.Close()
		}
	}()

	if err := g.encoder.Encode(resp); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}

	if err := g.encoder.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}

	return nil
}

type GobClientCodec struct {
	rwc    io.ReadWriteCloser
	dec    *gob.Decoder
	enc    *gob.Encoder
	encBuf *bufio.Writer
}

func NewGobClientCodec(conn io.ReadWriteCloser) *GobClientCodec {
	buf := bufio.NewWriter(conn)
	return &GobClientCodec{
		rwc:    conn,
		dec:    gob.NewDecoder(conn),
		enc:    gob.NewEncoder(buf),
		encBuf: buf,
	}
}

func (c *GobClientCodec) WriteRequest(r *RequestHeader, body any) (err error) {
	if err = c.enc.Encode(r); err != nil {
		return
	}
	if err = c.enc.Encode(body); err != nil {
		return
	}
	return c.encBuf.Flush()
}

func (c *GobClientCodec) ReadResponseHeader(r *ResponseHeader) error {
	return c.dec.Decode(r)
}

func (c *GobClientCodec) ReadResponseBody(body any) error {
	return c.dec.Decode(body)
}

func (c *GobClientCodec) Close() error {
	return c.rwc.Close()
}
