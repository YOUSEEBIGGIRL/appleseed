package loadbalance

import "testing"

func TestGet(t *testing.T) {
	countMap := make(map[string]int64)	// 记录每个 addr 被访问的次数
	addrs := []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}
	for _, v := range addrs {
		countMap[v] = 0
	}

	r := &RoundRobin{}
	r.SetAddrs(addrs)
	
	num := 50
	for i := 0; i < num; i++ {
		addr := r.Get()
		countMap[addr]++
	}
	
	t.Logf("result: %v\n", countMap)
}

func TestGetWithWeight(t *testing.T) {
	countMap := make(map[string]int64)
	addrs := []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}
	for _, v := range addrs {
		countMap[v] = 0
	}

	r := &RoundRobin{}
	r.SetAddrsWithWeight(map[string]int64{
		addrs[0]: 20,
		addrs[1]: 5,
		addrs[2]: 10,	
	})
	
	num := 50
	for i := 0; i < num; i++ {
		addr := r.GetWithWeight()
		countMap[addr]++
	}

	t.Logf("result: %v\n", countMap)
}
