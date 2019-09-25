package kissrpc

import (
	"encoding/gob"
	"errors"
	"log"
	"reflect"
)

const debug = false

//Types already known by gob.
var registeredTypes = map[string]struct{}{
	"string":     {},
	"float32":    {},
	"float64":    {},
	"byte":       {},
	"int":        {},
	"uint":       {},
	"int8":       {},
	"uint8":      {},
	"int16":      {},
	"uint16":     {},
	"int32":      {},
	"uint32":     {},
	"int64":      {},
	"uint64":     {},
	"error":      {},
	"bool":       {},
	"[]byte":     {},
	"complex64":  {},
	"complex128": {},
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

var gobErrorType = reflect.TypeOf(gobError{})

type gobError struct {
	Error string
}

func fromError(err error) gobError {
	return gobError{err.Error()}
}

func (this gobError) toError() error {
	return errors.New(this.Error)
}

func init() {
	RegisterType(gobError{})
}

func registerType(inType reflect.Type) {
	for inType.Kind() == reflect.Ptr {
		inType = inType.Elem()
	}
	err := registerInternal(inType, reflect.New(inType).Elem().Interface())
	if err != nil {
		return
	}
}

func registerInternal(t reflect.Type, v interface{}) error {
	if _, ok := registeredTypes[t.String()]; ok {
		return nil // Type already registered but its ok
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
			if t.Field(i).Name[0] >= 'a' && t.Field(i).Name[0] <= 'z' {
				continue
			}
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
