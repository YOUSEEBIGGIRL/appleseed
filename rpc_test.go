package appleseed

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
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
	s, err := NewServer(context.Background(), "service1", "127.0.0.1", "8880", reg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Register(new(XXX)); err != nil {
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

	//prefix := "/register-servier"
	addr, err := client.GetServerAddr(context.Background(), reg, &loadbalance.RoundRobin{}, "/register-servierservice1")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	cli := client.NewClient(conn, addr)

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
	reg, err := registry.NewEtcdClient([]string{"127.0.0.1:2379"})
	if err != nil {
		t.Fatal(err)
	}

	//prefix := "/register-servier"
	addr, err := client.GetServerAddr(context.Background(), reg, &loadbalance.RoundRobin{}, "/register-servier/service1")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	cli := client.NewClient(conn, addr)

	arg := &Args{RunTime: time.Minute}
	rep := &Reply{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	go func() {
		if err := cli.Call(ctx, "XXX.TimeoutFunc", arg, rep); err != nil {
			log.Fatal(err)
		}
	}()

	// 5s 后发送第二个 rpc 请求，此时 context 已经超时
	time.Sleep(time.Second * 5)
	go func() {
		if err := cli.Call(ctx, "XXX.TimeoutFunc", arg, rep); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(time.Minute)
}

func TestLoadBalanceServer(t *testing.T) {
	reg, err := registry.NewEtcd(context.Background(), []string{"127.0.0.1:2379"}, "", 5)
	if err != nil {
		t.Fatal("new etcd error: ", err)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// 模拟三个服务器提供同一个服务
	// simulate three servers to provide the same service
	go func() {
		defer wg.Done()
		s, err := NewServer(context.Background(), "service1", "127.0.0.1", "8880", reg)
		if err != nil {
			fmt.Printf("new server error: %v\n", err)
			return
		}
		if err := s.Register(new(XXX)); err != nil {
			fmt.Printf("register error: %v\n", err)
			return
		}
		if err := s.RunWithTCP(); err != nil {
			fmt.Printf("run server error: %v\n", err)
			return
		}
	}()

	go func() {
		defer wg.Done()
		s, err := NewServer(context.Background(), "service1", "127.0.0.1", "8880", reg)
		if err != nil {
			fmt.Printf("new server error: %v\n", err)
			return
		}
		if err := s.Register(new(XXX)); err != nil {
			fmt.Printf("register error: %v\n", err)
			return
		}
		if err := s.RunWithTCP(); err != nil {
			fmt.Printf("run server error: %v\n", err)
			return
		}
	}()

	go func() {
		defer wg.Done()
		s, err := NewServer(context.Background(), "service1", "127.0.0.1", "8880", reg)
		if err != nil {
			fmt.Printf("new server error: %v\n", err)
			return
		}
		if err := s.Register(new(XXX)); err != nil {
			fmt.Printf("register error: %v\n", err)
			return
		}
		if err := s.RunWithTCP(); err != nil {
			fmt.Printf("run server error: %v\n", err)
			return
		}
	}()

	wg.Wait()
}

func TestLoadBalanceClient(t *testing.T) {
	fn := func() {
		reg, err := registry.NewEtcdClient([]string{"127.0.0.1:2379"})
		if err != nil {
			t.Fatal(err)
		}

		//prefix := "/register-servier"
		addr, err := client.GetServerAddr(context.Background(), reg, &loadbalance.RoundRobin{}, "/register-servier/service1")
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("get server addr: %v\n", addr)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}

		cli := client.NewClient(conn, addr)
		arg := &Args{Str: "abc", X: 10, Y: 20}
		var reply Reply
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err := cli.Call(ctx, "XXX.Add", arg, &reply); err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v", reply)
	}

	for i := 0; i < 1000; i++ {
		fn()
	}
}

// 模拟大量连接，第一次可以运行，之后会阻塞，经过 debug 发现阻塞在了 client.GetServerAddr，进一步分析，发现是因为 etcd 挂掉了，
// 导致 client 从 etcd 中获取 key 阻塞，查看 etcd 日志，发现挂掉的原因是：open default.etcd/member/wal: too many open files
// 解决方法：修改系统的文件描述符最大数量，使用 ulimit -a 查看（未实践）
func TestRpcClientWithGoroutine(t *testing.T) {
	count := 3000
	var wg sync.WaitGroup
	wg.Add(count)
	// 10 个 goroutine，每个 goroutine 不断循环调用 rpc
	for i := 0; i < count; i++ {
		go func(i int) {
			defer wg.Done()
			reg, err := registry.NewEtcdClient([]string{"127.0.0.1:2379"})
			if err != nil {
				t.Error(err)
				return
			}

			//prefix := "/register-servier"
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			addr, err := client.GetServerAddr(ctx, reg, &loadbalance.RoundRobin{}, "/register-servier/service1")
			if err != nil {
				t.Error(err)
				return
			}

			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Error(err)
				return
			}

			cli := client.NewClient(conn, addr)
			ctx1, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			for j := 0; j < 10; j++ {
				arg := &Args{Str: "abc", X: int64(j), Y: int64(j)}
				var reply Reply
				if err := cli.Call(ctx1, "XXX.Add", arg, &reply); err != nil {
					log.Fatal(err)
				}
				t.Logf("[goroutine%d]%+v", i, reply)
			}
		}(i)
	}
	wg.Wait()
}
