package codec

type ServerCodec interface {
	ReadRequestHeader(header *RequestHeader) error
	ReadRequestBody(any) error
	WriteResponse(*ResponseHeader, any) error
	Close() error
}

type ClientCodec interface {
	WriteRequest(*RequestHeader, any) error
	ReadResponseHeader(header *ResponseHeader) error
	ReadResponseBody(any) error
	Close() error
}

type RequestHeader struct {
	ServiceMethod string
	Seq           uint64
}

func (r *RequestHeader) Reset() {
	r.Seq = 0
	r.ServiceMethod = ""
}

type ResponseHeader struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

func (r *ResponseHeader) Reset() {
	r.Seq = 0
	r.ServiceMethod = ""
	r.Error = ""
}
