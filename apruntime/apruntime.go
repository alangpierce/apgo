// The apruntime package contains all base operations.
package apruntime

import (
	"reflect"
	"go/token"
	"fmt"
)

type Context map[string]reflect.Value
type BuiltinFunc func (ctx Context, args []reflect.Value) reflect.Value

type NativePackage struct {
	Name string
	Funcs map[string]interface{}
	Globals map[string]*interface{}
}

func add(x interface{}, y interface{}) interface{} {
	// TODO: Handle other types.
	return reflect.ValueOf(x).Int() + reflect.ValueOf(y).Int()
}

var BinaryOperators = map[token.Token]reflect.Value{
	token.ADD: reflect.ValueOf(add),
}

var AssignBinaryOperators = map[token.Token]reflect.Value{
	token.ADD_ASSIGN: reflect.ValueOf(add),
}

var FmtPackage = &NativePackage{
	Name: "fmt",
	Funcs: map[string]interface{} {
		"Print": fmt.Print,
	},
	Globals: map[string]*interface{} {},
}