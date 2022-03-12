package appleseed

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/YOUSEEBIGGIRL/appleseed/client"
	"github.com/YOUSEEBIGGIRL/appleseed/loadbalance"
	"github.com/YOUSEEBIGGIRL/appleseed/registry"
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
	reg, err := registry.NewEtcd(context.Background(), []string{"127.0.0.1:2379"}, "", 5)
	if err != nil {
		t.Fatal("new etcd error: ", err)
	}
	s := NewServer("127.0.0.1", "8880", reg)
	if err := s.Register(context.Background(), new(XXX), "service1"); err != nil {
		t.Fatal(err)
	}
	if err := s.RunWithTCP(); err != nil {
		t.Fatal(err)
	}
}

func TestRpcClient(t *testing.T) {
	reg, err := registry.NewEtcdClient([]string{"127.0.0.1:2379"})
	if err != nil {
		t.Fatal(err)
	}

	prefix := "/register-servier"
	addr, err := client.GetServerAddr(context.Background(), reg, &loadbalance.RoundRobin{}, prefix, "service1")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	cli := client.NewClient(conn)
	arg := &Args{Str: "abc", X: 10, Y: 20}
	var reply Reply
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := cli.Call(ctx, "XXX.Add", arg, &reply); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", reply)
	
	// 调用不存在的方法
	if err := cli.Call(ctx, "XXX.Add1", arg, &reply); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", reply)
}

func TestClientTimeout(t *testing.T) {
	// arg := &Args{RunTime: time.Minute}
	// rep := &Reply{}
	// cli, err := Dial("tcp", ":9999")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	// defer cancel()

	// go func() {
	// 	if err := cli.Call(ctx, "XXX.TimeoutFunc", arg, rep); err != nil {
	// 		t.Fatal(err)
	// 	}
	// }()

	// // 5s 后发送第二个 rpc 请求，此时 context 已经超时
	// time.Sleep(time.Second * 5)
	// go func() {
	// 	if err := cli.Call(ctx, "XXX.TimeoutFunc", arg, rep); err != nil {
	// 		t.Fatal(err)
	// 	}
	// }()

	// time.Sleep(time.Minute)
}

func TestRpcClientWithGoroutine(t *testing.T) {
	// count := 300
	// var wg sync.WaitGroup
	// wg.Add(count)
	// // 10 个 goroutine，每个 goroutine 不断循环调用 rpc
	// for i := 0; i < count; i++ {
	// 	go func(i int) {
	// 		defer wg.Done()
	// 		cli, err := Dial("tcp", ":9999")
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		for j := 0; j < 1; j++ {
	// 			arg := &Args{Str: "abc", X: int64(j), Y: int64(j)}
	// 			var reply Reply
	// 			if err := cli.Call(context.Background(), "Cal.Add", arg, &reply); err != nil {
	// 				log.Fatal(err)
	// 			}
	// 			t.Logf("[goroutine%d]%+v", i, reply)
	// 		}
	// 	}(i)
	// }
	//wg.Wait()
}
