package loadbalance

type Balancer interface {
	// Addrs 保存所有的服务器地址
	Addrs() []string

	// AddrsWithWeight 保存所以的服务器地址以及对应的权重
	//AddrsWithWeight() map[string]int64

	// Get 均衡的从 addrs 中获取一个地址
	Get() string

	// GetWithWeight 均衡的从 AddrsWithWeight 中根据权重获取一个地址
	//GetWithWeight() string

	// SetAddrs 重置 addrs
	//SetAddrs(addrs []string)

	// SetAddrsWithWeight 重置 addrsWithWeight
	//SetAddrsWithWeight(addrsWithWeight map[string]int64)

	// Add 向负载均衡器中添加一个地址
	Add(addr string)

	// Update 更新负载均衡器中的一个地址
	Update(oldAddr string, newAddr string) error

	// Delete 删除负载均衡器中的一个地址
	Delete(addr string) error
}
