package registry

import (
	"context"
	"github.com/YOUSEEBIGGIRL/appleseed/loadbalance"
	"log"
	"sync"
	"testing"
	"time"
)

var registry *Etcd

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	r, err := NewEtcd(context.Background(), []string{"127.0.0.1:2379"}, "", 5)
	if err != nil {
		panic(err)
	}
	log.Println(r.lease)
	registry = r
}

func TestRegister(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := registry.Register(ctx, "service1", "127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ctx, "service1", "127.0.0.1:8081"); err != nil {
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
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := registry.Watch(context.Background(), "service1", &loadbalance.RoundRobin{}); err != nil {
			log.Fatalln(err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := registry.Register(ctx, "service1", "127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ctx, "service1", "127.0.0.1:8081"); err != nil {
		t.Fatal(err)
	}

	wg.Wait()
}
