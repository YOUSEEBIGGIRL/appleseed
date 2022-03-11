package loadbalance

type Balancer interface {
	// Addrs 保存所有的服务器地址
	Addrs() []string

	// AddrsWithWeight 保存所以的服务器地址以及对应的权重
	AddrsWithWeight() map[string]int64

	// Get 均衡的从 addrs 中获取一个地址
	Get() string

	// GetWithWeight 均衡的从 AddrsWithWeight 中根据权重获取一个地址
	GetWithWeight() string

	// SetAddrs 重置 addrs 
	SetAddrs(addrs []string)

	// SetAddrsWithWeight 重置 addrsWithWeight
	SetAddrsWithWeight(addrsWithWeight map[string]int64)
}