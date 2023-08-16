package loadbalance

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/autsu/rpcz/util"
)

var _ Interface = &RoundRobin{}
var _ Interface = &RoundRobinWithWeight{}

type RoundRobin struct {
	addrs []string
	// key 是地址，val 是该地址在 addrs 中的 index
	// 该字段用于 addrs 的更新和删除操作，使其时间复杂度为 O(1)
	addrsMap        map[string]int64
	addrsWithWeight map[string]*weightInfo
	idx             int64 // 用于记录轮询位置
}

func (r *RoundRobin) Get() (addr string) {
	if len(r.addrs) == 0 {
		return ""
	}
	l := len(r.addrs)
	if r.idx == int64(l) {
		r.idx = 0
	}
	addr = r.addrs[r.idx]
	r.idx++
	return
}

func (r *RoundRobin) Addrs() []string { return r.addrs }

func (r *RoundRobin) Add(addrs ...Addr) {
	if r.addrsMap == nil {
		r.addrsMap = make(map[string]int64)
	}
	for _, addr := range addrs {
		if addr.Addr == "" {
			continue
		}
		r.addrs = append(r.addrs, addr.Addr)
		r.addrsMap[addr.Addr] = int64(len(r.addrs) - 1)
	}
}

func (r *RoundRobin) Update(oldAddr Addr, newAddr Addr) error {
	if r.addrsMap == nil {
		r.addrsMap = make(map[string]int64)
	}
	index, ok := r.addrsMap[oldAddr.Addr]
	if !ok {
		errMsg := fmt.Errorf("not found %v", oldAddr)
		util.Log.Error("roundRobin.Update error", slog.String("err", errMsg.Error()))
		return errMsg
	}
	r.addrs[index] = newAddr.Addr
	delete(r.addrsMap, oldAddr.Addr)
	r.addrsMap[newAddr.Addr] = index
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

type RoundRobinWithWeight struct {
	rr RoundRobin

	addrsWithWeight map[string]*weightInfo
}

func (r *RoundRobinWithWeight) Update(oldAddr, newAddr Addr) error {
	if err := r.rr.Update(oldAddr, newAddr); err != nil {
		return err
	}
	delete(r.addrsWithWeight, oldAddr.Addr)
	r.addrsWithWeight[newAddr.Addr] = &weightInfo{
		addr:      newAddr.Addr,
		weight:    newAddr.Weight,
		curWeight: 0,
	}
	return nil
}

func (r *RoundRobinWithWeight) Delete(addr string) error {
	return r.rr.Delete(addr)
}

// weightInfo 平滑加权轮询需要该 struct 来保存一些信息
type weightInfo struct {
	addr      string
	weight    int64
	curWeight int64
}

func (r *RoundRobinWithWeight) Add(addrs ...Addr) {
	if r.addrsWithWeight == nil {
		r.addrsWithWeight = make(map[string]*weightInfo)
	}
	for _, addr := range addrs {
		r.addrsWithWeight[addr.Addr] = &weightInfo{weight: addr.Weight, addr: addr.Addr}
	}
}

// Get 使用平滑加权轮询算法获取地址
func (r *RoundRobinWithWeight) Get() (addr string) {
	var (
		total int64
		ret   *weightInfo
	)

	for _, wi := range r.addrsWithWeight {
		total += wi.weight
		wi.curWeight += wi.weight
		if ret == nil || wi.curWeight > ret.curWeight {
			ret = wi
		}
	}
	if ret == nil {
		return ""
	}
	ret.curWeight -= total
	return ret.addr
}
