package codec

import (
	"net"
	"testing"
	"time"
)

func TestGobWrite(t *testing.T) {
	var addr = ":7788"
	// start a server
	go func() {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			panic(err)
		}
		b := make([]byte, 4096)
		for {
			conn, err := lis.Accept()
			if err != nil {
				println(err)
				continue
			}
			for {
				_, err = conn.Read(b)
				if err != nil {
					println(err)
					continue
				}
				println(string(b))
			}
		}
	}()

	// Make sure the server runs before the client
	time.Sleep(time.Second * 3)

	// client: write data to conn
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	codec := NewGobClientCodec(conn)
	var writeFunc = func(c *GobClientCodec, r1, r2 any) error {
		if err = c.enc.Encode(r1); err != nil {
			return err
		}
		// sleep 3s，看看 c.enc.Encode 是立马写入，还是等到最后调用 c.encBuf.Flush()
		// 时一并写入，如果 server 那边是先输出 r1，再等待 3s 输出 r2，则代表是调用
		// c.enc.Encode 时会立马写入，如果最后在同一时间一起输出 r1 和 r2，则代表是
		// 调用 c.encBuf.Flush() 时才写入
		//
		// 测试结果：
		// 如果是 buf := bufio.NewWriter(conn); enc: gob.NewEncoder(buf)
		// 那么是调用 c.encBuf.Flush() 时才写入
		// 如果是 enc: gob.NewEncoder(conn)，那么在调用 c.enc.Encode 时立马写入
		time.Sleep(time.Second * 3)
		if err = c.enc.Encode(r2); err != nil {
			return err
		}
		return c.encBuf.Flush()
	}
	if err := writeFunc(codec, "123", "456"); err != nil {
		t.Fatal(err)
	}
	// Wait for a while, let the server output
	time.Sleep(time.Second * 3)
}
