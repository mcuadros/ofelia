package middlewares

import "reflect"

func IsEmpty(i interface{}) bool {
	t := reflect.TypeOf(i).Elem()
	e := reflect.New(t).Interface()

	return reflect.DeepEqual(i, e)
}
