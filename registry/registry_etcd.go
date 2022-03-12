package registry

import (
	"context"
	"log"
	"path"

	"github.com/google/uuid"
	client "go.etcd.io/etcd/client/v3"
)

const defaultServicePrefix = "/register-servier"

var _ Server = &Etcd{}
var _ Client = &Etcd{}

type Etcd struct {
	endpoints     []string
	conn          *client.Client
	prefix        string
	keepAliveTime int64
	lease         *client.LeaseGrantResponse
}

// NewRegistry 创建一个注册中心，etcdEndpoints 指定 etcd 的地址，prefix 表示公共前缀，
// 方便查找和分类，如果不指定则使用默认前缀，格式为：<prefix>/<serviceName>/<uuid>，
// keepAliveTimeout 表示超时时间，如果超过该时间没有发送心跳，则说明此服务器已下线
func NewEtcd(ctx context.Context, endpoints []string, prefix string, keepAliveTimeout int64) (r *Etcd, err error) {
	r = &Etcd{}
	r.endpoints = endpoints
	if prefix != "" {
		r.prefix = prefix
	} else {
		r.prefix = defaultServicePrefix // 默认前缀
	}
	r.keepAliveTime = keepAliveTimeout

	// 连接到 etcd
	c, err := client.New(client.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Println("connecting to etcd error: ", err)
		return nil, err
	}
	r.conn = c

	// 创建租约
	l := client.NewLease(c)
	lease, err := l.Grant(ctx, keepAliveTimeout)
	if err != nil {
		log.Println("grant lease error: ", err)
		return nil, err
	}
	r.lease = lease
	// 对租约进行永久保活
	l.KeepAlive(ctx, lease.ID)
	return
}

func NewEtcdClient(endpoints []string) (*Etcd, error) {
	var e Etcd
	// 连接到 etcd
	c, err := client.New(client.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Println("connecting to etcd error: ", err)
		return nil, err
	}
	e.conn = c
	return &e, nil
}

// Register 将服务注册到 etcd 中，以 serviceName 作为 key，对应的 addr 作为 val，同时会绑定租约来保持活性
func (e *Etcd) Register(ctx context.Context, serviceName, addr string) (err error) {
	// 一个 serviceName 可能由多台服务器提供，用一个 uuid 来唯一标识一台服务器，查找时
	// 将 serviceName 作为前缀查找即可
	key := path.Join(e.prefix, serviceName, uuid.NewString())
	// 写入 kv 到 etcd 并绑定租约
	_, err = e.conn.Put(ctx, key, addr, client.WithLease(e.lease.ID))
	if err != nil {
		log.Printf("register service[%v] error: %v\n", serviceName, err)
		return
	}
	log.Printf("register a service, key: %v\n", key)
	return
}

// Unregister 将已注册的服务从 etcd 中移除
func (e *Etcd) Unregister(ctx context.Context, serviceName string) (err error) {
	_, err = e.conn.Delete(ctx, path.Join(e.prefix, serviceName))
	return
}

func (e *Etcd) Name() string {
	return "etcd"
}

func (e *Etcd) Addr() []string {
	return e.endpoints
}

// Get 从 etcd 中通过 serviceName 获取该 service 的所有地址，需要客户端自己做负载均衡
func (e *Etcd) Get(ctx context.Context, serviceName string) (addrs []string, err error) {
	gr, err := e.conn.Get(ctx, path.Join(e.prefix, serviceName), client.WithPrefix())
	for _, v := range gr.Kvs {
		addrs = append(addrs, string(v.Value))
	}
	return
}
