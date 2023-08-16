package loadbalance

type Interface interface {
	// Addrs 保存所有的服务器地址
	// 好像没啥卵用
	//Addrs() []string

	// Get 均衡的从 addrs 中获取一个地址
	Get() string

	// Add 向负载均衡器中添加地址
	Add(addr ...Addr)

	// Update 更新负载均衡器中的一个地址
	Update(oldAddr Addr, newAddr Addr) error

	// Delete 删除负载均衡器中的一个地址
	Delete(addr string) error
}

type Addr struct {
	Addr   string
	Weight int64
}
