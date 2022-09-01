package appleseed

import (
	"github.com/autsu/appleseed/codec"
	"sync"
)

var RequestPool = &requestPool{
	p: &sync.Pool{New: func() any { return &codec.RequestHeader{} }},
}

var ResponsePool = &responsePool{
	p: &sync.Pool{New: func() any { return &codec.ResponseHeader{} }},
}

type requestPool struct {
	p *sync.Pool
}

func (p *requestPool) Malloc() *codec.RequestHeader {
	return p.p.Get().(*codec.RequestHeader)
}

// Free 清空对象 r，然后 put 到 sync.pool
func (p *requestPool) Free(r *codec.RequestHeader) {
	r.Reset()
	p.p.Put(r)
}

type responsePool struct {
	p *sync.Pool
}

func (p *responsePool) Malloc() *codec.ResponseHeader {
	return p.p.Get().(*codec.ResponseHeader)
}

func (p *responsePool) Free(r *codec.ResponseHeader) {
	p.p.Put(r)
}
