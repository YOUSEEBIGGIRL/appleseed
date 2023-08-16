package loadbalance

import "testing"

func TestRoundRobinGet(t *testing.T) {
	r := &RoundRobin{}
	countMap := make(map[string]int64) // 记录每个 addr 被访问的次数
	addrs := []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}
	for _, addr := range addrs {
		countMap[addr] = 0
		r.Add(Addr{Addr: addr})
	}

	num := 50
	for i := 0; i < num; i++ {
		addr := r.Get()
		countMap[addr]++
	}

	t.Logf("result: %v\n", countMap)
}

func TestRoundRobinCRUD(t *testing.T) {
	r := &RoundRobin{}

	checkAddrExistInSlice := func(addr string) bool {
		for _, v := range r.addrs {
			if addr == v {
				return true
			}
		}
		return false
	}

	checkAddrExistInMap := func(addr string) bool {
		_, ok := r.addrsMap[addr]
		return ok
	}

	addrs := []Addr{
		{Addr: "127.0.0.1"},
		{Addr: "192.168.1.1"},
		{Addr: "10.0.0.1"},
	}
	r.Add(addrs...)

	// 没看懂这个有什么意义
	//a := r.Addrs()
	//a = append(a, "123") // 不会破坏封装性
	//t.Log(r.addrs)

	oldAddr := "10.0.0.1"
	newAddr := "10.0.0.2"
	err := r.Update(Addr{Addr: oldAddr}, Addr{Addr: newAddr})
	if err != nil {
		t.Errorf("update error: %v", err)
		return
	}

	if checkAddrExistInMap(oldAddr) || checkAddrExistInSlice(oldAddr) {
		t.Errorf("%s want not exist but got exist", oldAddr)
		return
	}

	if !checkAddrExistInSlice(newAddr) || !checkAddrExistInSlice(newAddr) {
		t.Errorf("%s want exist but got not exist", newAddr)
		return
	}

	deleteAddr := "10.0.0.2"
	err = r.Delete(deleteAddr)
	if err != nil {
		t.Errorf("delete error: %v", err)
		return
	}

	if checkAddrExistInMap(deleteAddr) || checkAddrExistInSlice(deleteAddr) {
		t.Errorf("%s want not exist but got exist", deleteAddr)
		return
	}
}

func TestGetWithWeight(t *testing.T) {
	countMap := make(map[string]int64)
	addrs := []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}
	for _, v := range addrs {
		countMap[v] = 0
	}

	r := &RoundRobinWithWeight{}
	r.Add([]Addr{
		{Addr: addrs[0], Weight: 20},
		{Addr: addrs[1], Weight: 5},
		{Addr: addrs[2], Weight: 10},
	}...)

	num := 50
	for i := 0; i < num; i++ {
		addr := r.Get()
		countMap[addr]++
	}

	t.Logf("result: %v\n", countMap)

	// 按照权重，被访问次数由大到小的排名应该为：addrs[0] > addrs[2] > addrs[1]
	if !(countMap[addrs[1]] < countMap[addrs[2]] && countMap[addrs[2]] < countMap[addrs[0]]) {
		t.Errorf("error, want count is addrs[0] > addrs[2] > addrs[1]")
		return
	}
}
