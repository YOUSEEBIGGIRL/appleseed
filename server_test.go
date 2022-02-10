package appleseed

import (
	"log"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func TestServer(t *testing.T) {
	server := NewServer()
	if err := server.Register(new(Cal)); err != nil {
		log.Fatalln(err)
	}
	if err := server.RunWithTCP("localhost", "8080"); err != nil {
		log.Fatalln(err)
	}
}

//func TestClient(t *testing.T) {
//	conn, err := net.Dial("tcp", ":8080")
//	if err != nil {
//		log.Fatalln(err)
//	}
//
//	c := NewGobCodec(conn)
//
//	h := &Header{
//		ServiceMethod: "Cal.Add",
//		Seq:           uint64(1),
//	}
//	//log.Println(h)
//	c.Write(h, fmt.Sprintf("rpc req %d", h.Seq))
//
//	c.ReadHeader(h)
//	var reply string
//	c.ReadBody(reply)
//	log.Println("reply: ", reply)
//}
