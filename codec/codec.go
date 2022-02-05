package codec

type Codec interface {
	ReadRequestHeader(*RequestHeader) error
	ReadRequestBody(interface{}) error
	WriteResponse(*ResponseHeader, interface{}) error
}

type RequestHeader struct {
	ServiceMethod string
	Seq           uint64
}

type ResponseHeader struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}
