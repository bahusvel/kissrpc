package kissrpc

import (
	"encoding/gob"
	"errors"
	"log"
	"reflect"
)

const debug = false

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
	err := registerInternal(inType, reflect.Indirect(reflect.New(inType)).Interface())
	if err != nil {
		return
	}
}

func registerInternal(t reflect.Type, v interface{}) error {
	if _, ok := registeredTypes[t.String()]; ok {
		return errors.New("Type is already registered")
	}
	switch t.Kind() {
	case reflect.Interface:
		return errors.New("Type is an interface, please register structs implementing this interface directly instead")
	case reflect.Func:
		panic("Type is a function, functions cannot be registered as values")
	case reflect.Chan:
		panic("Type is a channel, channels cannot be registered as values")
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			registerType(t.Field(i).Type)
		}
	}
	if debug {
		log.Println("Registering type", t.String())
	}
	gob.Register(v)
	registeredTypes[t.String()] = struct{}{}
	return nil
}

func RegisterType(regType interface{}) {
	t := reflect.TypeOf(regType)
	err := registerInternal(t, regType)
	if err != nil {
		panic(err)
	}
}

func registerInsOuts(f reflect.Type) {
	if f.Kind() != reflect.Func {
		panic("f is not a function")
	}
	for i := 0; i < f.NumIn(); i++ {
		registerType(f.In(i))
	}
	for i := 0; i < f.NumOut(); i++ {
		registerType(f.Out(i))
	}
}
