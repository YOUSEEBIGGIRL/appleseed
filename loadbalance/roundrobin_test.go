package loadbalance

import "testing"

func TestGet(t *testing.T) {
	r := &RoundRobin{}
	countMap := make(map[string]int64) // 记录每个 addr 被访问的次数
	addrs := []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}
	for _, v := range addrs {
		countMap[v] = 0
		r.Add(v)
	}

	num := 50
	for i := 0; i < num; i++ {
		addr := r.Get()
		countMap[addr]++
	}

	t.Logf("result: %v\n", countMap)
}

func Test(t *testing.T) {
	r := &RoundRobin{}
	addrs := []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}
	for _, v := range addrs {
		r.Add(v)
	}
	a := r.Addrs()
	a = append(a, "123") // 不会破坏封装性
	t.Log(r.addrs)

	err := r.Update("10.0.0.1", "10.0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r.Addrs())

	err = r.Delete("10.0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r.Addrs())
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
