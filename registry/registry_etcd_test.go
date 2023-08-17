package registry

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/autsu/rpcz"
	"github.com/autsu/rpcz/loadbalance"
)

var registry *Etcd

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	r, err := NewEtcd(context.Background(), []string{"127.0.0.1:2379"}, 5)
	if err != nil {
		panic(err)
	}
	log.Println(r.lease)
	registry = r
}

func newFakeServer(serviceName string) *rpcz.Server {
	return rpcz.NewServer(serviceName, "localhost", "8090")
}

func TestRegister(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := registry.Register(ctx, newFakeServer("test1")); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ctx, newFakeServer("test2")); err != nil {
		t.Fatal(err)
	}
}

func TestGet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	addrs, err := registry.Get(ctx, "service1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(addrs)
}

func TestWatch(t *testing.T) {
	ctx := context.TODO()
	testSvcName := "test111"

	// 注册一个服务，并将地址添加到 lb
	if err := registry.Register(ctx, rpcz.NewServer(testSvcName, "localhost", "8090")); err != nil {
		t.Fatal(err)
	}
	t.Log("services: ", registry.services)

	addrs, err := registry.Get(ctx, testSvcName)
	if err != nil {
		t.Fatal(err)
	}

	lb := loadbalance.RoundRobin{}
	lb.Add(loadbalance.ConvStrAddrsToAddrs(addrs)...)
	t.Log(lb.Addrs())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		registry.Watch(context.Background(), &lb)
	}()

	// test watch
	func() {
		// register a new addr
		if err := registry.Register(ctx, rpcz.NewServer(testSvcName, "localhost", "8080")); err != nil {
			t.Fatal(err)
		}
		t.Log(lb.Addrs())
	}()

	wg.Wait()
}
