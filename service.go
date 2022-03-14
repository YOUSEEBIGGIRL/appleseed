package appleseed

import (
	"github.com/YOUSEEBIGGIRL/appleseed/codec"
	"log"
	"reflect"
	"sync"
)

// service 可以理解为是一个对象，它的方法被会被注册到 rpc 中，客户可以调用通过 "对象.方法"
// 的形式来进行 rpc 调用
type service struct {
	name    string
	typ     reflect.Type
	val     reflect.Value
	methods map[string]*MethodInfo
}

type MethodInfo struct {
	sync.Mutex
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	callNum   uint64
}

func (s *service) call(srv *Server, sendLock *sync.Mutex, wg *sync.WaitGroup, method *MethodInfo, c codec.ServerCodec, req *codec.RequestHeader, argv, replyv reflect.Value) {
	if wg != nil {
		defer wg.Done()
	}
	method.Lock()
	method.callNum++
	method.Unlock()

	returnValues := method.method.Func.Call([]reflect.Value{s.val, argv, replyv})
	log.Println("after call, reply value: ", replyv.Interface())
	var errMsg string
	errRet := returnValues[0].Interface()
	if errRet != nil {
		errMsg = errRet.(error).Error()
	}
	srv.sendResponse(sendLock, req, c, replyv.Interface(), errMsg)
	req.Reset()
	srv.reqPool.Put(req)
}
