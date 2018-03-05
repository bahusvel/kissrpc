package kissrpc

import (
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
)

type Client struct {
	connection net.Conn
	encoder    *gob.Encoder
	decoder    *gob.Decoder
}

func ConnectService(conn net.Conn, service interface{}) error {
	client, err := NewClient(conn)
	if err != nil {
		return err
	}
	val := reflect.ValueOf(service)
	if val.Kind() != reflect.Ptr || reflect.Indirect(val).Kind() != reflect.Struct {
		panic(fmt.Errorf("Supplied service is not a pointer to struct"))
	}
	val = reflect.Indirect(val)
	serviceType := val.Type()
	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		if field.Type.Kind() != reflect.Func {
			panic(fmt.Errorf("Field %s of %s is not a function, only functions are supported", field.Name, serviceType.Name()))
		}
		fieldFunc := reflect.MakeFunc(field.Type, func(in []reflect.Value) []reflect.Value {
			rets, err := client.valueCall(serviceType.Name()+"."+field.Name, in)
			if err != nil {
				panic(err)
			}
			return rets
		})
		val.Field(i).Set(fieldFunc)
	}
	return nil
}

func NewClient(conn net.Conn) (*Client, error) {
	client := Client{conn, gob.NewEncoder(conn), gob.NewDecoder(conn)}
	return &client, nil
}

func (this Client) valueCall(name string, in []reflect.Value) ([]reflect.Value, error) {
	args := []interface{}{}
	for _, inarg := range in {
		args = append(args, inarg.Interface())
	}

	err := this.encoder.Encode(call{Name: name, Args: args})
	if err != nil {
		return []reflect.Value{}, err
	}

	retValues := callReturn{}
	err = this.decoder.Decode(&retValues)
	if err != nil {
		return []reflect.Value{}, err
	}
	rets := []reflect.Value{}
	for _, retval := range retValues.ReturnValues {
		rets = append(rets, reflect.ValueOf(retval))
	}
	return rets, err
}

func (this Client) Call(name string, args ...interface{}) ([]interface{}, error) {
	err := this.encoder.Encode(call{Name: name, Args: args})
	if err != nil {
		return []interface{}{}, err
	}
	retValues := callReturn{}
	err = this.decoder.Decode(&retValues)
	if err != nil {
		return []interface{}{}, err
	}
	rets := retValues.ReturnValues
	if len(rets) != 0 {
		if err, ok := rets[len(rets)-1].(error); ok {
			return rets[:len(rets)-1], err
		}
	}
	return rets, nil
}

func (this Client) Call1(name string, args ...interface{}) (interface{}, error) {
	var rets []interface{}
	var err error
	rets, err = this.Call(name, args...)
	if err != nil {
		return []interface{}{}, err
	}
	if len(rets) != 1 {
		return []interface{}{}, fmt.Errorf("Unexpected return values for %s expected %d got %d", name, 1, len(rets))
	}
	return rets[0], err
}
func (this Client) Call2(name string, args ...interface{}) (interface{}, interface{}, error) {
	var rets []interface{}
	var err error
	rets, err = this.Call(name, args...)
	if len(rets) != 2 {
		return []interface{}{}, []interface{}{}, fmt.Errorf("Unexpected return values for %s expected %d got %d", name, 2, len(rets))
	}
	return rets[0], rets[1], err
}
