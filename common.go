package kissrpc

import (
	"encoding/gob"
	"log"
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

type call struct {
	Name  string
	Args  []interface{}
	Async bool
}

type callReturn struct {
	ReturnValues []interface{}
	Error        error
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
	t := reflect.TypeOf(regType)
	if _, ok := registeredTypes[t.String()]; ok {
		return
	}
	if t.Kind() == reflect.Interface { // Interfaces should not be registered
		return
	}
	if DEBUG {
		log.Println("Registering type", t.String())
	}
	gob.Register(regType)
	registeredTypes[t.String()] = struct{}{}
}

func RegisterNamedType(name string, regType interface{}) {
	t := reflect.TypeOf(regType)
	if _, ok := registeredTypes[name]; ok {
		return
	}
	if t.Kind() == reflect.Interface { // Interfaces should not be registered
		return
	}
	if DEBUG {
		log.Println("Registering type", t.String())
	}
	gob.RegisterName(name, regType)
	registeredTypes[name] = struct{}{}

}
