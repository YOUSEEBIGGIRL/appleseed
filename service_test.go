package appleseed

import (
	"testing"
)

func TestRegisterStruct(t *testing.T) {
	new(Server).Register(new(XXX))
}

func TestCall(t *testing.T) {
	//sev := new(Server)
	//if err := sev.Register(new(Cal)); err != nil {
	//	log.Fatalln(err)
	//}
	//
	//p := &Param{
	//	Str: "123",
	//	X:   10,
	//	Y:   20,
	//}
	//
	//r := new(Res)
	//
	//if err := sev.call("Cal.Add", p, r); err != nil {
	//	log.Fatalln(err)
	//}
	//
	//log.Println("result: ", r)
}
