package money

import "reflect"

func GoValidatorCustomTypeFunc(field reflect.Value) interface{} {
	if value, ok := field.Interface().(Money); ok {
		return value.Float64()
	}
	return nil
}
