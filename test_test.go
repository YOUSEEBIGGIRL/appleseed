package appleseed

import (
	"log"
	"sync"
	"testing"
)

func TestRpcServer(t *testing.T) {
	s := NewServer()
	if err := s.Register(new(Cal)); err != nil {
		t.Fatal(err)
	}
	if err := s.RunWithTCP("localhost", "9999"); err != nil {
		t.Fatal(err)
	}
}

func TestRpcClient(t *testing.T) {
	cli, err := Dial("tcp", ":9999")
	if err != nil {
		t.Fatal(err)
	}
	arg := &Param{Str: "abc", X: 10, Y: 20}
	var reply Res
	if err := cli.Call("Cal.Add", arg, &reply); err != nil {
		t.Fatal(err)
	}
	// 调用不存在的方法
	if err := cli.Call("Cal.Add1", arg, &reply); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", reply)
}

func TestRpcClientWithGoroutine(t *testing.T) {
	count := 30
	var wg sync.WaitGroup
	wg.Add(count)
	// 10 个 goroutine，每个 goroutine 不断循环调用 rpc
	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()
			cli, err := Dial("tcp", ":9999")
			if err != nil {
				log.Fatal(err)
			}
			for j := 0; j < 10; j++ {
				arg := &Param{Str: "abc", X: int64(j), Y: int64(j)}
				var reply Res
				if err := cli.Call("Cal.Add", arg, &reply); err != nil {
					log.Fatal(err)
				}
				t.Logf("[goroutine%d]%+v", i, reply)
			}
		}(i)
	}
	wg.Wait()
}
