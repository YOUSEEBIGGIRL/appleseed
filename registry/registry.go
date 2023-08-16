package registry

import (
	"context"

	"github.com/autsu/rpcz"
)

type Server interface {
	// Name 返回注册中心名字（比如 Etcd）
	Name() string

	// Addr 返回注册中心的地址（可能有多个）
	Addr() []string

	// Register 注册 serviceName 到注册中心
	Register(ctx context.Context, server *rpcz.Server) error

	// Unregister 从注册中心中删除 serviceName
	Unregister(ctx context.Context, serviceName string) (err error)
}

type Client interface {
	// Get 从注册中心中获取 serviceName 对应的 address
	Get(ctx context.Context, serviceName string) (addrs []string, err error)
}
