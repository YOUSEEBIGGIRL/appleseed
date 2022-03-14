package client

import (
	"context"
	"github.com/YOUSEEBIGGIRL/appleseed/loadbalance"
	"github.com/YOUSEEBIGGIRL/appleseed/registry"
	"log"
	"testing"
)

func TestGetServerAddr(t *testing.T) {
	reg, err := registry.NewEtcdClient([]string{"127.0.0.1:2379"})
	if err != nil {
		t.Fatal(err)
	}

	prefix := "/register-servier"

	for i := 0; i < 1000; i++ {
		addr, err := GetServerAddr(context.Background(), reg, &loadbalance.RoundRobin{}, prefix, "service1")
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("get server addr: %v\n", addr)
	}

}
