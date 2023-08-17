package loadbalance

func ConvStrAddrsToAddrs(addr []string) []Addr {
	ret := make([]Addr, 0, len(addr))
	for _, s := range addr {
		ret = append(ret, Addr{Addr: s})
	}
	return ret
}
