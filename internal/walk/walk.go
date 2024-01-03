/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package walk

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
)

type WalkFunc func(x any, path []string, tag reflect.StructTag) error

func Walk(x any, f WalkFunc) error {
	v, ok := x.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(x)
	}
	if v.Kind() != reflect.Pointer || v.IsNil() {
		panic("non-nil pointer expected")
	}
	errs := walk(v, nil, "", f)
	if len(errs) > 0 {
		return multierror.Append(nil, errs...)
	}
	return nil
}

type walkError struct {
	err  error
	path []string
}

func (e walkError) Error() string {
	return fmt.Sprintf("/%s: %s", strings.Join(e.path, "/"), e.err)
}

func walk(v reflect.Value, path []string, tag reflect.StructTag, f WalkFunc) (errs []error) {
	t := v.Type()

	callback := func() {
		var x any
		if t.Kind() != reflect.Pointer && v.CanAddr() {
			x = v.Addr().Interface()
		} else {
			x = v.Interface()
		}
		if err := f(x, path, tag); err != nil {
			errs = append(errs, walkError{err: err, path: path})
		}
	}

	switch t.Kind() {
	case reflect.String,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		callback()
	case reflect.Slice, reflect.Array:
		callback()
		for i := 0; i < v.Len(); i++ {
			errs = append(errs, walk(v.Index(i), append(path, strconv.Itoa(i)), tag, f)...)
		}
	case reflect.Map:
		callback()
		for it := v.MapRange(); it.Next(); {
			errs = append(errs, walk(it.Value(), append(path, it.Key().String()), tag, f)...)
		}
	case reflect.Struct:
		callback()
		for _, field := range reflect.VisibleFields(t) {
			if field.IsExported() {
				errs = append(errs, walk(v.FieldByIndex(field.Index), append(path, field.Name), field.Tag, f)...)
			}
		}
	case reflect.Pointer:
		if v.IsNil() {
			callback()
		} else {
			errs = append(errs, walk(v.Elem(), path, tag, f)...)
		}
	case reflect.Interface:
		if v.IsNil() {
			callback()
		} else {
			errs = append(errs, walk(v.Elem(), path, tag, f)...)
		}
	default:
		panic(walkError{err: fmt.Errorf("unrecognized type: %v", t.Kind()), path: path})
	}
	return
}
