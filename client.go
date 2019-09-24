package kissrpc

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
)

func init() {
	RegisterType(map[string]string{})
}

type Client struct {
	Conn    net.Conn
	encoder *gob.Encoder
	decoder *gob.Decoder
	callMutex sync.Mutex
}

var errorType = reflect.TypeOf(new(error)).Elem()

func genZeroReturn(f reflect.Type) (ret []reflect.Value) {
	if f.Kind() != reflect.Func {
		panic("f is not a function")
	}
	for i := 0; i < f.NumOut(); i++ {
		ret = append(ret, reflect.New(f.Out(i)).Elem())
	}
	return ret
}

func (this *Client) MakeProxyFunc(name string, PtrToFunc interface{}) error {
	val := reflect.ValueOf(PtrToFunc)
	if val.Kind() != reflect.Ptr || reflect.Indirect(val).Kind() != reflect.Func {
		panic(fmt.Errorf("Supplied service is not a pointer to struct"))
	}
	ret, err := this.Call("kissrpc.getTable")
	if err != nil {
		return err
	}
	if len(ret) != 1 {
		return errors.New("Cannot get method table from server")
	}
	table, ok := ret[0].(map[string]string)
	if !ok {
		return errors.New("Cannot get method table from server")
	}

	if v, ok := table[name]; !ok || v != val.Type().String() {
		return fmt.Errorf("Requested method %s was not found on the server or its signature does not match", name)
	}

	this.makeProxyFunc(name, val)

	return nil
}

func (this *Client) makeProxyFunc(name string, val reflect.Value) {
	t := val.Type()
	registerInsOuts(t)
	async := t.NumOut() == 0
	lastIsError := !async && t.Out(t.NumOut()-1) == errorType
	zeroReturn := []reflect.Value{}
	if lastIsError {
		zeroReturn = genZeroReturn(t)
	}
	val.Set(reflect.MakeFunc(val.Type(), func(in []reflect.Value) []reflect.Value {
		rets, err := this.valueCall(name, in, async)
		if err != nil {
			if lastIsError {
				rets = zeroReturn
				rets[len(rets)-1] = reflect.ValueOf(&err).Elem()
			} else {
				panic(err)
			}
		}
		if lastIsError && !rets[len(rets)-1].IsValid() {
			// prevents invalid error value if nil, HACK I think this is still problematic if return value contains any other interface whose value is nil.
			rets[len(rets)-1] = reflect.Zero(errorType)
		}
		return rets
	}))
}

func (this *Client) MakeService(service interface{}) error {
	val := reflect.ValueOf(service)
	if val.Kind() != reflect.Ptr || reflect.Indirect(val).Kind() != reflect.Struct {
		panic(fmt.Errorf("Supplied service is not a pointer to struct"))
	}

	ret, err := this.Call("kissrpc.getTable")
	if err != nil {
		return err
	}
	if len(ret) != 1 {
		return errors.New("Cannot get method table from server")
	}
	table, ok := ret[0].(map[string]string)
	if !ok {
		return errors.New("Cannot get method table from server")
	}

	val = reflect.Indirect(val)
	serviceType := val.Type()
	for i := 0; i < serviceType.NumField(); i++ {
		field := serviceType.Field(i)
		if field.Type.Kind() != reflect.Func {
			panic(fmt.Errorf("Field %s of %s is not a function, only functions are permitted for declaration of services", field.Name, serviceType.Name()))
		}

		fieldName := serviceType.Name() + "." + field.Name

		if v, ok := table[fieldName]; !ok || v != field.Type.String() {
			return fmt.Errorf("Requested method %s was not found on the server or its signature does not match", fieldName)
		}

		this.makeProxyFunc(fieldName, val.Field(i))
	}
	return nil
}

func NewClient(conn net.Conn) *Client {
	client := Client{Conn: conn, encoder: gob.NewEncoder(conn), decoder: gob.NewDecoder(conn)}
	return &client
}

func (this *Client) valueCall(name string, in []reflect.Value, async bool) ([]reflect.Value, error) {
	this.callMutex.Lock()
	defer this.callMutex.Unlock()
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
		val := reflect.ValueOf(retval)
		if val.IsValid() && val.Type() == gobErrorType {
			realError := retValues.ReturnValues[len(retValues.ReturnValues)-1].(gobError).toError()
			val = reflect.ValueOf(&realError).Elem()
		}
		rets = append(rets, val)
	}
	return rets, err
}

func (this *Client) Call(name string, args ...interface{}) ([]interface{}, error) {
	this.callMutex.Lock()
	defer this.callMutex.Unlock()
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

func (this *Client) AsyncCall(name string, args ...interface{}) (encode_err error) {
	this.callMutex.Lock()
	defer this.callMutex.Unlock()
	encode_err = this.encoder.Encode(call{Name: name, Args: args, Async: true})
	return
}

func (this *Client) Call1(name string, args ...interface{}) (interface{}, error) {
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

func (this *Client) Call2(name string, args ...interface{}) (interface{}, interface{}, error) {
	var rets []interface{}
	var err error
	rets, err = this.Call(name, args...)
	if len(rets) != 2 {
		return []interface{}{}, []interface{}{}, fmt.Errorf("Unexpected return values for %s expected %d got %d", name, 2, len(rets))
	}
	return rets[0], rets[1], err
}
