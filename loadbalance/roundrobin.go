package loadbalance

import (
	"fmt"
	"log"
)

var addrsIndex int64 // 用于轮询

type RoundRobin struct {
	addrs []string
	// key 是地址，val 是该地址在 addrs 中的 index，该字段用于 addrs 的更新和删除操作
	addrsMap        map[string]int64
	addrsWithWeight map[string]*weightInfo
}

// weightInfo 平滑加权轮询需要该 struct 来保存一些信息
type weightInfo struct {
	addr      string
	weight    int64
	curWeight int64
}

func (r *RoundRobin) Get() (addr string) {
	if len(r.addrs) == 0 {
		return ""
	}
	l := len(r.addrs)
	if addrsIndex == int64(l) {
		addrsIndex = 0
	}
	addr = r.addrs[addrsIndex]
	addrsIndex++
	return
}

// GetWithWeight 使用平滑加权轮询算法
func (r *RoundRobin) GetWithWeight() (addr string) {
	var (
		total     int64
		retStruct *weightInfo
	)

	for _, wi := range r.addrsWithWeight {
		total += wi.weight
		wi.curWeight += wi.weight
		if retStruct == nil || wi.curWeight > retStruct.curWeight {
			retStruct = wi
		}
	}
	if retStruct == nil {
		return ""
	}
	retStruct.curWeight -= total
	return retStruct.addr
}

func (r *RoundRobin) Addrs() []string {
	return r.addrs
}

func (r *RoundRobin) AddrsWithWeight() (m map[string]int64) {
	m = make(map[string]int64)
	for k, v := range r.addrsWithWeight {
		m[k] = v.weight
	}
	return
}

// SetAddrs 设置 addrs
//func (r *RoundRobin) SetAddrs(addrs []string) {
//	r.addrs = addrs
//}

// SetAddrsWithWeight 设置 addrsWithWeight
func (r *RoundRobin) SetAddrsWithWeight(addrsWithWeight map[string]int64) {
	if r.addrsWithWeight == nil {
		r.addrsWithWeight = make(map[string]*weightInfo)
	}
	for addr, weight := range addrsWithWeight {
		r.addrsWithWeight[addr] = &weightInfo{weight: weight, addr: addr}
	}
}

func (r *RoundRobin) Add(addr string) {
	if r.addrsMap == nil {
		r.addrsMap = make(map[string]int64)
	}
	r.addrs = append(r.addrs, addr)
	r.addrsMap[addr] = int64(len(r.addrs) - 1)
}

func (r *RoundRobin) Update(oldAddr string, newAddr string) error {
	if r.addrsMap == nil {
		r.addrsMap = make(map[string]int64)
	}
	index, ok := r.addrsMap[oldAddr]
	if !ok {
		log.Printf("not found %v", oldAddr)
		return fmt.Errorf("not found %v", oldAddr)
	}
	r.addrs[index] = newAddr
	delete(r.addrsMap, oldAddr)
	r.addrsMap[newAddr] = index
	return nil
}

func (r *RoundRobin) Delete(addr string) error {
	if r.addrsMap == nil {
		r.addrsMap = make(map[string]int64)
	}
	index, ok := r.addrsMap[addr]
	if !ok {
		log.Printf("not found %v", addr)
		return fmt.Errorf("not found %v", addr)
	}
	r.addrs = append(r.addrs[:index], r.addrs[index+1:]...)
	delete(r.addrsMap, addr)
	return nil
}
