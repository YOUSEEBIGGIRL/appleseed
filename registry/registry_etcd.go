package registry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"path"

	"github.com/autsu/rpcz"
	"github.com/autsu/rpcz/loadbalance"
	"github.com/autsu/rpcz/util"

	"github.com/google/uuid"
	client "go.etcd.io/etcd/client/v3"
)

const servicePrefix = "/__rpcz-register-servier__"

var _ Server = &Etcd{}
var _ Client = &Etcd{}

type Etcd struct {
	endpoints     []string
	conn          *client.Client
	keepAliveTime int64
	lease         *client.LeaseGrantResponse
	services      []string
}

// NewEtcd 创建一个注册中心，etcdEndpoints 指定 etcd 的地址，prefix 表示公共前缀，
// 方便查找和分类，如果不指定则使用默认前缀，格式为：<prefix>/<serviceName>/<uuid>，
// keepAliveTimeout 表示超时时间，如果超过该时间没有发送心跳，则说明此服务器已下线
func NewEtcd(ctx context.Context, endpoints []string, keepAliveTimeout int64) (r *Etcd, err error) {
	r = &Etcd{}
	r.endpoints = endpoints
	r.keepAliveTime = keepAliveTimeout

	// 连接到 etcd
	c, err := client.New(client.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		util.Log.Error("connecting to etcd error", slog.Any("error", err))
		return nil, err
	}
	r.conn = c

	// 创建租约
	l := client.NewLease(c)
	lease, err := l.Grant(ctx, keepAliveTimeout)
	if err != nil {
		util.Log.Error("grant lease error", slog.Any("error", err))
		return nil, err
	}
	r.lease = lease
	// 对租约进行永久保活
	ch, err := l.KeepAlive(ctx, lease.ID)
	if err != nil {
		util.Log.Error("set keepalive error", slog.Any("error", err))
		return nil, err
	}
	go func() {
		// keepAlive 的 response 需要消费掉，不然 etcd 日志会一直警告：keepalive 缓存队列已满，新的 response 将被丢弃
		for range ch {
		}
	}()

	return
}

func NewEtcdClient(endpoints []string) (*Etcd, error) {
	var e Etcd
	// 连接到 etcd
	c, err := client.New(client.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		util.Log.Error("connecting to etcd error", slog.Any("err", err))
		return nil, err
	}
	e.conn = c
	return &e, nil
}

// Register 将服务注册到 etcd 中，以 serviceName 作为 key，对应的 addr 作为 val，同时会绑定租约来保持存活性
func (e *Etcd) Register(ctx context.Context, server *rpcz.Server) (err error) {
	if server == nil {
		return errors.New("server is nil")
	}

	// 一个 serviceName 可能由多台服务器提供，用一个 uuid 来唯一标识一台服务器，查找时
	// 将 serviceName 作为前缀查找即可
	key := path.Join(servicePrefix, server.ServiceName(), uuid.NewString())

	// 写入 kv 到 etcd 并绑定租约
	_, err = e.conn.Put(ctx, key, server.Addr(), client.WithLease(e.lease.ID))
	if err != nil {
		util.Log.Error("etcd.Register register service error: ", slog.Any("err", err), slog.String("serviceName", server.ServiceName()))
		return
	}

	util.Log.Debug("register new service",
		slog.String("service name", server.ServiceName()),
		slog.String("registration center info", fmt.Sprintf("name: %v, addr: %v", e.Name(), e.Addr())),
		slog.String("registry key/value info", fmt.Sprintf("key: %v, value: %v", server.ServiceName(), e.Addr())),
	)

	e.services = append(e.services, server.ServiceName())
	//go func() {
	//	if err_ := e.Watch(context.Background(), serviceName, &loadbalance.RoundRobin{}); err_ != nil {
	//		err = err_
	//		return
	//	}
	//}()
	return
}

// Unregister 将已注册的服务从 etcd 中移除
func (e *Etcd) Unregister(ctx context.Context, serviceName string) (err error) {
	_, err = e.conn.Delete(ctx, path.Join(servicePrefix, serviceName))
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
	gr, err := e.conn.Get(ctx, path.Join(servicePrefix, serviceName), client.WithPrefix())
	for _, v := range gr.Kvs {
		addrs = append(addrs, string(v.Value))
	}
	return
}

// Watch TODO: 需要重新设计，lo 这个参数不合理
// 废弃
func (e *Etcd) __Watch(ctx context.Context, serviceName string, lo loadbalance.Interface) error {
	watchChan := e.conn.Watch(ctx, path.Join(servicePrefix, serviceName), client.WithPrefix(), client.WithPrevKV())
	for {
		select {
		case resp := <-watchChan:
			for _, event := range resp.Events {
				switch event.Type {
				case client.EventTypePut:
					if event.IsCreate() { // 新的 key
						log.Printf("watch a new key[key=%s, val=%s] put\n", event.Kv.Key, event.Kv.Value)
						//lo.Add(string(event.Kv.Value))
					} else if event.IsModify() { // 已存在的 key 的 val 发生了变化
						log.Printf("watch a key update[key=%s, new val=%s]\n", event.Kv.Key, event.Kv.Value)
						//if err := lo.Update(string(event.PrevKv.Value), string(event.Kv.Value)); err != nil {
						//	return err
						//}
					}
				case client.EventTypeDelete:
					log.Printf("watch a key[key=%s, val=%s] delete\n", event.Kv.Key, event.Kv.Value)
					if err := lo.Delete(string(event.Kv.Value)); err != nil {
						return err
					}
				}
			}
		}
	}
}

func (e *Etcd) Watch1(ctx context.Context, handleEvent func(event *client.Event)) error {
	var err error
	for _, service := range e.services {
		service := service
		go func() {
			watchChan := e.conn.Watch(ctx, path.Join(servicePrefix, service), client.WithPrefix(), client.WithPrevKV())
			for {
				select {
				case <-ctx.Done():
					err = ctx.Err()
					return
				case resp := <-watchChan:
					for _, event := range resp.Events {
						handleEvent(event)
					}
				}
			}
		}()
	}
	return err
}

// Watch watch key 并更新 loadbalance 信息
func (e *Etcd) Watch(ctx context.Context, lb loadbalance.Interface) {
	for _, service := range e.services {
		service := service
		go func() {
			watchChan := e.conn.Watch(ctx, path.Join(servicePrefix, service), client.WithPrefix(), client.WithPrevKV())
			for {
				select {
				case <-ctx.Done():
					util.Log.Warn("watch finish, because ctx.Done", "ctx.Err", ctx.Err())
					return
				case resp := <-watchChan:
					for _, event := range resp.Events {
						switch event.Type {
						case client.EventTypePut:
							if event.IsCreate() { // 新的 key
								util.Log.Info("create event", slog.String("key", string(event.Kv.Key)), slog.String("val", string(event.Kv.Value)))
								lb.Add(loadbalance.Addr{Addr: string(event.Kv.Value)})
							} else if event.IsModify() { // 已存在的 key 的 val 发生了变化
								util.Log.Info("update event", slog.String("key", string(event.Kv.Key)), slog.String("val", string(event.Kv.Value)))
								// 这里的错误不关键，可以忽略
								lb.Update(loadbalance.Addr{Addr: string(event.PrevKv.Value)}, loadbalance.Addr{Addr: string(event.Kv.Value)})
							}
						case client.EventTypeDelete:
							util.Log.Info("delete event", slog.String("key", string(event.Kv.Key)), slog.String("val", string(event.Kv.Value)))
							// 同 update，非关键性错误
							lb.Delete(string(event.Kv.Value))
						}
					}
				}
			}
		}()
	}
}
