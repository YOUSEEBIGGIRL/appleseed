package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCoder struct {
	conn    io.ReadWriteCloser // 通常是通过 TCP 或者 Unix 建立 socket 时得到的连接
	buf     *bufio.Writer      // 包装 socket conn
	decoder *gob.Decoder       // gob 解码
	encoder *gob.Encoder       // gob 编码
	closed  bool               // 防止重复关闭
}

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCoder{
		conn:    conn,
		buf:     buf,
		decoder: gob.NewDecoder(conn), // 从 conn 中读取数据，并用 gob 解析出来
		encoder: gob.NewEncoder(buf),  // 将数据写入到 buf 中，并用 gob 编码数据
	}
}

func (g *GobCoder) Close() error {
	if g.closed {
		return nil
	}
	g.closed = true
	return g.conn.Close()
}

// ReadRequestHeader 从 conn 的数据中，使用 gob 解析出 header 部分
func (g *GobCoder) ReadRequestHeader(req *RequestHeader) error {
	return g.decoder.Decode(req)
}

// ReadRequestBody 从 conn 的数据中，使用 gob 解析出 body 部分
func (g *GobCoder) ReadRequestBody(body interface{}) error {
	return g.decoder.Decode(body)
}

// Write 使用 gob 对 head 和 body 进行编码，并写入到 conn 中
func (g *GobCoder) WriteResponse(resp *ResponseHeader, body interface{}) error {
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
