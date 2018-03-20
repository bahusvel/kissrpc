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

var errorType = reflect.TypeOf(new(error)).Elem()

func genZeroReturn(f reflect.Type) (ret []reflect.Value) {
	if f.Kind() != reflect.Func {
		panic("f is not a function")
	}
	for i := 0; i < f.NumOut(); i++ {
		ret = append(ret, reflect.New(f.Out(i)))
	}
	return ret
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
			panic(fmt.Errorf("Field %s of %s is not a function, only functions are permitted for declaration of services", field.Name, serviceType.Name()))
		}

		lastIsError := false
		if field.Type.NumOut() != 0 && field.Type.Out(field.Type.NumOut()-1) == errorType {
			lastIsError = true
		}

		fieldFunc := reflect.MakeFunc(field.Type, func(in []reflect.Value) []reflect.Value {
			var rets []reflect.Value
			var err error
			if field.Type.NumOut() == 0 {
				// Async
				rets, err = client.valueCall(serviceType.Name()+"."+field.Name, in, true)
				if err != nil {
					panic(err)
				}

			} else {
				rets, err = client.valueCall(serviceType.Name()+"."+field.Name, in, false)
				if err != nil {
					if lastIsError {
						rets = genZeroReturn(field.Type)
						rets[len(rets)-1] = reflect.ValueOf(&err).Elem()
					} else {
						panic(err)
					}
				}
				if lastIsError && !rets[len(rets)-1].IsValid() {
					// prevents invalid error value if nil, HACK I think this is still problematic of return value contains any other interface whose value is nil.
					rets[len(rets)-1] = reflect.Zero(errorType)
				}
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

func (this Client) valueCall(name string, in []reflect.Value, async bool) ([]reflect.Value, error) {
	args := []interface{}{}
	for _, inarg := range in {
		args = append(args, inarg.Interface())
	}

	err := this.encoder.Encode(call{Name: name, Args: args, Async: async})
	if err != nil {
		return []reflect.Value{}, err
	}

	if async {
		return []reflect.Value{}, nil
	}
	retValues := callReturn{}
	err = this.decoder.Decode(&retValues)
	if err != nil {
		return []reflect.Value{}, err
	}
	if retValues.Error != nil {
		return []reflect.Value{}, retValues.Error
	}
	rets := []reflect.Value{}
	for _, retval := range retValues.ReturnValues {
		rets = append(rets, reflect.ValueOf(retval))
	}
	return rets, err
}

func (this Client) Call(name string, args ...interface{}) ([]interface{}, error) {
	err := this.encoder.Encode(call{Name: name, Args: args, Async: false})
	if err != nil {
		return []interface{}{}, err
	}
	retValues := callReturn{}
	err = this.decoder.Decode(&retValues)
	if err != nil {
		return []interface{}{}, err
	}
	if retValues.Error != nil {
		return []interface{}{}, retValues.Error
	}
	rets := retValues.ReturnValues
	if len(rets) != 0 {
		if err, ok := rets[len(rets)-1].(error); ok {
			return rets[:len(rets)-1], err
		}
	}
	return rets, nil
}

func (this Client) AsyncCall(name string, args ...interface{}) (encode_err error) {
	encode_err = this.encoder.Encode(call{Name: name, Args: args, Async: true})
	return
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
