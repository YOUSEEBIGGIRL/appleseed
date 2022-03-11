package appleseed

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"
)

type Args struct {
	Str     string
	X, Y    int64
	RunTime time.Duration
}

type Reply struct {
	Str string
	Add int64
}

type XXX struct{}

func (x *XXX) Add(args *Args, reply *Reply) error {
	reply.Add = args.X + args.Y
	reply.Str = args.Str
	return nil
}

func (x *XXX) TimeoutFunc(args *Args, reply *Reply) error {
	time.Sleep(args.RunTime)
	reply.Str = "DONE."
	return nil
}

func TestRpcServer(t *testing.T) {
	s := NewServer()
	if err := s.Register(new(XXX)); err != nil {
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
	arg := &Args{Str: "abc", X: 10, Y: 20}
	var reply Reply
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := cli.Call(ctx, "Cal.Add", arg, &reply); err != nil {
		t.Fatal(err)
	}
	// 调用不存在的方法
	if err := cli.Call(ctx, "Cal.Add1", arg, &reply); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", reply)
}

func TestClientTimeout(t *testing.T) {
	arg := &Args{RunTime: time.Minute}
	rep := &Reply{}
	cli, err := Dial("tcp", ":9999")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	go func() {
		if err := cli.Call(ctx, "XXX.TimeoutFunc", arg, rep); err != nil {
			t.Fatal(err)
		}
	}()

	// 5s 后发送第二个 rpc 请求，此时 context 已经超时
	time.Sleep(time.Second * 5)
	go func() {
		if err := cli.Call(ctx, "XXX.TimeoutFunc", arg, rep); err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Minute)
}

func TestRpcClientWithGoroutine(t *testing.T) {
	count := 300
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
			for j := 0; j < 1; j++ {
				arg := &Args{Str: "abc", X: int64(j), Y: int64(j)}
				var reply Reply
				if err := cli.Call(context.Background(), "Cal.Add", arg, &reply); err != nil {
					log.Fatal(err)
				}
				t.Logf("[goroutine%d]%+v", i, reply)
			}
		}(i)
	}
	wg.Wait()
}
