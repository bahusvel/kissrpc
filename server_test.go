package kissrpc

import (
	"log"
	"net"
	"testing"
)

type TestService struct {
	Hello func()
	Error func() error
}

func TestSimpleCall(t *testing.T) {
	s, c := net.Pipe()
	mtable := MethodTable{}
	mtable.AddFunc("Test", func(text string, number int) {
		log.Println("Hello!", text, number)
	})
	server := NewServer(s, mtable)
	go server.Serve()

	client, err := NewClient(c)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = client.Call("Test", "Test", 1)
	if err != nil {
		t.Error(err.Error())
	}
	server.Stop()
}

func TestSimpleService(t *testing.T) {
	s, c := net.Pipe()
	mtable := MethodTable{}
	mtable.AddService(TestService{Hello: func() { log.Println("Hello") }, Error: func() error { return nil }})
	server := NewServer(s, mtable)
	go server.Serve()

	clientService := TestService{}
	err := ConnectService(c, &clientService)
	if err != nil {
		log.Fatal(err)
	}
	clientService.Hello()
	clientService.Error()
	server.Stop()
}

type TestIface interface {
	is_testiface()
}

type TestStruct struct {
	Text string
}

func (this TestStruct) is_testiface() {

}

func TestInterfaceFunc(t *testing.T) {
	s, c := net.Pipe()
	mtable := MethodTable{}
	RegisterType(TestStruct{})
	mtable.AddFunc("Test", func(test TestIface) {
		log.Println("Hello!", test.(TestStruct).Text)
	})
	server := NewServer(s, mtable)
	go server.Serve()

	client, err := NewClient(c)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = client.Call("Test", TestIface(TestStruct{"Test"}))
	if err != nil {
		t.Error(err.Error())
	}
	server.Stop()
}

func BenchmarkSimpleCall(b *testing.B) {
	s, c := net.Pipe()
	mtable := MethodTable{}
	mtable.AddFunc("Test", func(number int) int {
		//log.Println("Hello!", text, number)
		return number
	})
	server := NewServer(s, mtable)
	go server.Serve()

	client, err := NewClient(c)
	if err != nil {
		b.Error(err.Error())
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		client.Call("Test", n)
	}
	//server.Stop()
}
