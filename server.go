package appleseed

import (
	"context"
	"errors"
	"fmt"
	"go/token"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"

	"github.com/YOUSEEBIGGIRL/appleseed/codec"
	"github.com/YOUSEEBIGGIRL/appleseed/registry"
	reuseport "github.com/kavu/go_reuseport"
)

// 发生错误时，将该空结构体作为 body 发送
var (
	invalidRequest = struct{}{}
	typeOfError    = reflect.TypeOf((*error)(nil)).Elem()
)

type Server struct {
	registerService sync.Map
	sendMu          sync.Mutex
	wg              sync.WaitGroup
	reqPool         *sync.Pool
	respPool        *sync.Pool
	reg             registry.Server
	addr            string
}

func NewServer(host, port string, reg registry.Server) *Server {
	if reg == nil {
		panic("register is nil")
	}
	s := &Server{}
	s.reg = reg
	s.reqPool = &sync.Pool{New: func() any { return &codec.RequestHeader{} }}
	s.respPool = &sync.Pool{New: func() any { return &codec.ResponseHeader{} }}
	s.addr = fmt.Sprintf("%s:%s", host, port)
	return s
}

func (s *Server) Register(ctx context.Context, struct_ any, serviceName string) error {
	// 检查 struct_ 是否是一个 struct
	kind := reflect.TypeOf(struct_).Kind()
	if kind == reflect.Ptr {
		if reflect.TypeOf(struct_).Elem().Kind() != reflect.Struct {
			return errors.New("register error: param is not a struct type")
		}
	} else if reflect.TypeOf(struct_).Kind() != reflect.Struct {
		return errors.New("register error: param is not a struct type")
	}

	stype := reflect.TypeOf(struct_)
	sval := reflect.ValueOf(struct_)
	sname := reflect.Indirect(sval).Type().Name()
	// struct 必须可导出
	if !token.IsExported(sname) {
		errMsg := fmt.Sprintf("rpc.Register: type %v is not exported", stype.String())
		log.Println(errMsg)
		return errors.New(errMsg)
	}
	// struct 不能是匿名结构体
	if sname == "" {
		errMsg := "rpc.Register: no service name for type " + stype.String()
		log.Println(errMsg)
		return errors.New(errMsg)
	}

	srv := new(service)
	srv.typ = stype
	srv.val = sval
	srv.name = sname
	srv.methods = suitableMethods(stype)
	if len(srv.methods) == 0 {
		str := ""
		// To help the user, see if a pointer receiver would work.
		method := suitableMethods(reflect.PtrTo(stype))
		if len(method) != 0 {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
		} else {
			str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
		}
		log.Print(str)
		return errors.New(str)
	}

	if _, ok := s.registerService.LoadOrStore(sname, srv); ok {
		return errors.New("该结构体已经注册")
	}
	// 同时添加到注册中心
	s.reg.Register(ctx, serviceName, s.addr)
	log.Printf("register [structName:%s, serviceName:%s] to [%s:%v] success, register info: [key:%v value: %v]\n",
		sname, serviceName, s.reg.Name(), s.reg.Addr(), serviceName, s.addr)
	return nil
}

// suitableMethods 获取 typ 下的所有合法方法
func suitableMethods(typ reflect.Type) map[string]*MethodInfo {
	methods := make(map[string]*MethodInfo)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if !method.IsExported() {
			continue
		}
		mt := method.Type
		mname := method.Name
		paramNum := mt.NumIn()
		// 标准格式的 func 需要有三个参数：接收者，request，response
		if paramNum != 3 {
			log.Printf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mname, paramNum)
			continue
		}
		// 标准格式的 func 需要有一个 error 类型的返回值
		returnNum := mt.NumOut()
		if returnNum != 1 {
			log.Printf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mname, returnNum)
			continue
		}

		argType := mt.In(1)
		// 第一个参数必须可导出
		if !isExportedOrBuiltinType(argType) {
			log.Printf("rpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
			continue
		}

		// 标准格式的 func 第二个参数（response）必须为指针类型
		replyType := mt.In(2)
		if replyType.Kind() != reflect.Ptr {
			log.Printf("rpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
			continue
		}
		// 第二个参数必须可导出
		if !isExportedOrBuiltinType(replyType) {
			continue
		}
		// 标准格式的 func 返回值必须为 error 类型
		returnType := mt.Out(0)
		if returnType != typeOfError {
			log.Printf("rpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
			continue
		}
		log.Printf("rpc.Register: method name: %v\n", mname)
		methods[mname] = &MethodInfo{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
	}
	return methods
}

// isExportedOriBuiltinType 检测 t 是否是可导出类型或者基本类型？
func isExportedOrBuiltinType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath 返回包名，代表这个包的唯一标识符，所以可能是单一的包名，或者 encoding/base64。
	// 对于 Go 内置的类型 string,error 等，或者未定义名称的类型 struct{} 等，则返回空字符串。
	// 自定义类型的 PkgPath 不为空
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *Server) RunWithTCP() error {
	listen, err := reuseport.Listen("tcp", s.addr)
	//listen, err := net.Listen("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return err
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go s.serverConn(conn)
	}
}

func (s *Server) serverConn(conn net.Conn) {
	c := codec.NewGobServerCodec(conn)
	s.ServerCodec(c)
}

// ServerCodec 使用长连接的方式来处理 client 的请求
func (s *Server) ServerCodec(c codec.ServerCodec) {
	sendLock := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		// 读取 request
		service, mtype, req, argv, replyv, keepReading, err := s.readRequest(c)
		if err != nil {
			if err != io.EOF {
				log.Println("rpc: ", err)
			}
			// keepReading 为 false 时，说明 err 为 EOF，即对方已断开连接
			if !keepReading {
				break
			}
			if req != nil {
				// 回应错误信息
				s.sendResponse(sendLock, req, c, invalidRequest, err.Error())
				req.Reset()
				s.reqPool.Put(req)
			}
			continue
		}
		wg.Add(1)
		go service.call(s, sendLock, wg, mtype, c, req, argv, replyv)
	}
	wg.Wait()
	c.Close()
}

func (s *Server) readRequestHeader(c codec.ServerCodec) (svc *service, mtype *MethodInfo, req *codec.RequestHeader, keepReading bool, err error) {
	req = s.reqPool.Get().(*codec.RequestHeader)
	var errMsg string
	if err = c.ReadRequestHeader(req); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			errMsg = fmt.Sprintf("rcp server: read header error: %v", err.Error())
			log.Println(errMsg)
			return
		}
		// 如果读取错误为 EOF，说明对方已断开连接，此时 keepReading 返回 false，使得最外层的
		// serverCodec 的 for {} 可以被终止
		return nil, nil, nil, false, errors.New(errMsg)
	}
	log.Printf("request head: %+v \n", req)

	// 从这里开始产生的错误属于非严重错误，比如用户传入的 serviceName 格式错误、service 未找到、
	// method 未找到，这些错误对整个系统影响并不大，所以可以跳过该请求，继续处理该连接上的下个请求
	keepReading = true
	dot := strings.LastIndex(req.ServiceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc: service/method request ill-formed: " + req.ServiceMethod)
		return
	}
	// 解析出服务名和方法名
	serviceName := req.ServiceMethod[:dot]
	methodName := req.ServiceMethod[dot+1:]
	ser, ok := s.registerService.Load(serviceName)
	if !ok {
		err = errors.New("rpc: can't find service " + req.ServiceMethod)
		return
	}
	svc = ser.(*service)
	// 获取方法的相关信息
	mtype = svc.methods[methodName]
	if mtype == nil {
		err = errors.New("rpc: can't find method " + req.ServiceMethod)
	}
	return
}

func (s *Server) readRequest(c codec.ServerCodec) (service *service, mtype *MethodInfo, req *codec.RequestHeader, argv, replyv reflect.Value, keepReading bool, err error) {
	service, mtype, req, keepReading, err = s.readRequestHeader(c)
	if err != nil {
		// discard body
		// 虽然发生了错误，但是仍然需要将连接中的剩余数据（body）消费掉
		// 如果 gob.Decode() 传入的是 nil，那么 gob 会读取连接中的一个值并
		// 将该值丢弃，比如 conn 中使用 gob 序列化了 a，b 两个对象，此时
		// 第一次 decode(nil)，那么 gob 将从 conn 中读取 a 并将其丢弃，
		// 第二次 decode(&b)，gob 会读取下一个值 b
		c.ReadRequestBody(nil)
		return
	}

	var isValue bool
	// 构造 arg 和 reply
	if mtype.ArgType.Kind() == reflect.Ptr {
		// reflect.New() 会创建一个表示指向指定类型的新零值的指针
		// 比如如果传入的 reflect.Type 是 int64，那么会返回一个 *int64
		// 如果传入的是 *int64，那么会返回一个 **int64
		// 如果传入的是指针类型，那么需要使用 Elem 获取原始类型，否则会创建一个二级指针
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		isValue = true
	}

	if err = c.ReadRequestBody(argv.Interface()); err != nil {
		log.Println("rpc server: read argv err: ", err)
	}
	// 如果用户传入的 argv 是值类型
	if isValue {
		// 因为 argv 是 reflect.New 创建出来的，是一个指针，所以如果用户传入的是值类型,
		// 那么就同样需要构造一个值类型参数，使用 Elem 来取得指针对应的值
		argv = argv.Elem()
	}

	// reply 必须是一个指针类型
	replyv = reflect.New(mtype.ReplyType.Elem())
	switch mtype.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mtype.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mtype.ReplyType.Elem(), 0, 0))
	}
	return
}

func (s *Server) sendResponse(sendLock *sync.Mutex, req *codec.RequestHeader, c codec.ServerCodec, reply interface{}, errMsg string) {
	respHeader := s.respPool.Get().(*codec.ResponseHeader)
	respHeader.ServiceMethod = req.ServiceMethod
	respHeader.Seq = req.Seq
	if errMsg != "" {
		respHeader.Error = errMsg
		reply = invalidRequest
	}
	// 加锁的作用？
	// 不加锁，偶尔 client 会出现 read response header error:  EOF
	// 从 github 上找到一个答案：发现这里加锁是为了避免缓冲区 c.buf.Flush() 的时候，其他
	// goroutine 也在往同一个缓冲区写入，从而导致 err: short write 的错误
	// 听起来挺有道理的，但是我测试没有出现过这个错误，而是 EOF
	sendLock.Lock()
	if err := c.WriteResponse(respHeader, reply); err != nil {
		log.Println("rpc server: write response err: ", err)
	}
	sendLock.Unlock()
	log.Println("send ok")
	// 重新放到对象池中复用
	respHeader.Reset()
	s.respPool.Put(respHeader)
}
