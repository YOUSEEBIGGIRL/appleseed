package appleseed

import (
	"context"
	"log"
	"testing"
	"time"
)

var registry *Registry

func init() {
	r, err := NewRegistry(context.Background(), []string{"127.0.0.1:2379"}, "", 5)
	if err != nil {
		panic(err)
	}
	log.Println(r.lease)
	registry = r
}

func TestRegister(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	if err := registry.Register(ctx, "service1", "127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(ctx, "service1", "127.0.0.1:8081"); err != nil {
		t.Fatal(err)
	}
}

func TestGet(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	addrs, err := registry.Get(ctx, "service1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(addrs)
}
