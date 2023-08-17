## Usage Example:


struct:
```go
package api 

import (
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
```

server:
```go
package main

import (
	"context"
	"log"
	
	"/path/to/api"
	
	"github.com/autsu/rpcz"
	"github.com/autsu/rpcz/registry"
)

var serviceName = "service1"

func main() {
	ctx := context.Background()
	reg, err := registry.NewEtcd(ctx, []string{"127.0.0.1:2379"}, 5)
	if err != nil {
		log.Fatal("new etcd error: ", err)
	}

	s := rpcz.NewServer(serviceName, "127.0.0.1", "8880")

	if err := reg.Register(ctx, s); err != nil {
		log.Fatal(err)
	}

	if err := s.Register(new(api.XXX)); err != nil {
		log.Fatal(err)
	}
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
```

client:
```go
package main

import (
	"context"
	"log"
	"net"

	"/path/to/api"

	"github.com/autsu/rpcz"
	"github.com/autsu/rpcz/registry"
	"github.com/autsu/rpcz/loadbalance"
	"github.com/autsu/rpcz/client"
)

var serviceName = "service1"

func main() {
	ctx := context.Background()
	reg, err := registry.NewEtcdClient([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	lb := &loadbalance.RoundRobin{}
	go reg.Watch(ctx, lb)

	addr, err := client.GetServerAddr(ctx, reg, lb, serviceName)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	cli := client.NewClient(conn, addr)

	arg := &api.Args{Str: "abc", X: 10, Y: 20}
	var reply api.Reply
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := cli.Call(ctx, "XXX.Add", arg, &reply); err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", reply)
}
```