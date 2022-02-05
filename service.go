package easyrpc

import (
	"errors"
	"log"
	"reflect"
	"strings"
	"sync"
)

type service struct {
	registerStruct sync.Map
}

func (s *service) Register(obj interface{}) error {
	kind := reflect.TypeOf(obj).Kind()
	if kind == reflect.Ptr {
		if reflect.TypeOf(obj).Elem().Kind() != reflect.Struct {
			panic("传入的参数不是一个结构体")
		}
	} else if reflect.TypeOf(obj).Kind() != reflect.Struct {
		panic("传入的参数不是一个结构体")
	}


	val := reflect.ValueOf(obj)

	structName := reflect.Indirect(reflect.ValueOf(obj)).Type().Name()
	if structName == "" {
		return errors.New("TODO")
	}

	if _, ok := s.registerStruct.LoadOrStore(structName, val); ok {
		return errors.New("该结构体已经注册")
	}

	log.Println(structName, "register ok")

	return nil
}

func (s *service) call(service string, param, res interface{}) error {
	index := strings.Index(service, ".")
	if index == -1 {
		return errors.New("wrong service, format: [struct.func]")
	}

	obj := service[:index]
	fn := service[index+1:]

	log.Println(obj, fn)
	load, ok := s.registerStruct.Load(obj)
	if !ok {
		return errors.New("not found this obj, you need register first")
	}

	val, ok := load.(reflect.Value)
	if !ok {
		return errors.New("val type is not reflect.Value")
	}

	result := val.
		MethodByName(fn).
		Call([]reflect.Value{reflect.ValueOf(param), reflect.ValueOf(res)})
	if err := result[0].Interface(); err != nil {
		return errors.New("call func error: " + err.(error).Error())
	}

	return nil
}
