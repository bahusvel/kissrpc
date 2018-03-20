package kissrpc

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
)

func init() {
	RegisterType([]interface{}{})
}

type MethodTable map[string]reflect.Value

type Server struct {
	Conn        net.Conn
	encoder     *gob.Encoder
	decoder     *gob.Decoder
	methodTable MethodTable
}

func NewServer(conn net.Conn, mtable MethodTable) *Server {
	server := &Server{
		conn,
		gob.NewEncoder(conn),
		gob.NewDecoder(conn),
		mtable,
	}
	server.methodTable.AddFunc("kissrpc.getTable", server.getTable)
	return server
}

func (this *Server) Stop() {
	this.Conn.Close()
}

func (this *Server) Serve() {
	for {
		callRequest := call{}
		ret := callReturn{}

		err := this.decoder.Decode(&callRequest)
		if err != nil {
			log.Println("Failed to read call request", err)
			return
		}
		var method reflect.Value
		var ok bool
		if debug {
			log.Println("Calling", callRequest.Name)
		}
		if method, ok = this.methodTable[callRequest.Name]; !ok {
			log.Println("Requested method not found", callRequest.Name)
			ret.Error = errors.New("Requested method not found")
			return
		}
		arguments := []reflect.Value{}
		for _, arg := range callRequest.Args {
			arguments = append(arguments, reflect.ValueOf(arg))
		}
		for _, retVal := range method.Call(arguments) {
			ret.ReturnValues = append(ret.ReturnValues, retVal.Interface())
		}
		if callRequest.Async {
			continue
		}
		this.encoder.Encode(ret)
		if err != nil {
			log.Println("Failed to write call response", err)
			return
		}
	}
}

func (this MethodTable) AddFunc(name string, function interface{}) {
	val := reflect.ValueOf(function)
	if val.Kind() != reflect.Func {
		panic(fmt.Errorf("%s is not a function", name))
	}
	funcType := val.Type()
	if debug {
		log.Printf("Function %s has type %s\n", name, funcType.String())
	}
	for i := 0; i < funcType.NumIn(); i++ {
		registerType(funcType.In(i))
	}
	for i := 0; i < funcType.NumOut(); i++ {
		registerType(funcType.Out(i))
	}
	this[name] = val
}

func (this MethodTable) AddService(service interface{}) {
	val := reflect.ValueOf(service)
	if val.Kind() != reflect.Struct {
		panic(fmt.Errorf("Supplied service is not a struct"))
	}
	serviceType := val.Type()
	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		if field.Type.Kind() != reflect.Func {
			panic(fmt.Errorf("Field %s of %s is not a function, only functions are supported", field.Name, serviceType.Name()))
		}
		this.AddFunc(serviceType.Name()+"."+field.Name, val.Field(i).Interface())
	}
}

func (this Server) getTable() map[string]string {
	wiretable := map[string]string{}
	for k, v := range this.methodTable {
		wiretable[k] = v.Type().String()
	}
	return wiretable
}
