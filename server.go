package kissrpc

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"reflect"
)

const DEBUG = false

var registeredTypes = map[string]struct{}{
	"string": {},
	"float":  {},
	"int":    {},
	"int32":  {},
	"int64":  {},
	"error":  {},
}

func init() {
	RegisterType([]interface{}{})
}

type call struct {
	Name string
	Args []interface{}
}

type callReturn struct {
	ReturnValues []interface{}
	Error        error
}

type MethodTable map[string]reflect.Value

type Server struct {
	connection  net.Conn
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
	return server
}

func (this *Server) Stop() {
	this.connection.Close()
}

func (this *Server) Serve() {
	for {
		callRequest := call{}
		err := this.decoder.Decode(&callRequest)
		if err != nil {
			log.Println("Failed to read call request", err)
			return
		}
		var method reflect.Value
		var ok bool
		if DEBUG {
			log.Println("Calling", callRequest.Name)
		}
		if method, ok = this.methodTable[callRequest.Name]; !ok {
			log.Println("Requested method not found", callRequest.Name)
			return
		}

		arguments := []reflect.Value{}
		for _, arg := range callRequest.Args {
			arguments = append(arguments, reflect.ValueOf(arg))
		}

		retVals := []interface{}{}
		for _, retVal := range method.Call(arguments) {
			retVals = append(retVals, retVal.Interface())
		}
		this.encoder.Encode(callReturn{ReturnValues: retVals})
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
	if DEBUG {
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

func registerType(inType reflect.Type) {
	if _, ok := registeredTypes[inType.String()]; ok {
		return
	}
	if inType.Kind() == reflect.Interface { // Interfaces should not be registered
		return
	}
	if DEBUG {
		log.Println("Registering type", inType.String())
	}
	gob.Register(reflect.Indirect(reflect.New(inType)).Interface())
	registeredTypes[inType.String()] = struct{}{}

}

func RegisterType(regType interface{}) {
	gob.Register(regType)
}

func RegisterNamedType(name string, regType interface{}) {
	gob.RegisterName(name, regType)
}
