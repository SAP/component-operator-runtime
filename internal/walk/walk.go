/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
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

// Walk through x recursively (using reflection), and apply f to each node. In detail, this means:
//   - slices,arrays and maps will be traversed element by element (note that the order is not predictable in case of maps)
//   - structs will be traversed field by field (only exported fields).
//
// The walk function will be supplied with the following input:
//   - pointer nodes will be passed as such
//   - non-pointer nodes which are addressable will be passed as pointer to the node
//   - other nodes (e.g. non-pointer map entries) will be passed as such
//   - path is a string slice describing the path in the tree from root to node; map keys and struct fields as they are,
//     slice/array indices converted to string
//   - tag is the struct tag of the most recent struct field seen while traversing.
//
// Notes:
//   - Walk may panic in certain situations, e.g. if it is passed non-pointer or nil pointer, or if an unsupported value is
//     encountered while traversing (e.g. a channel)
//   - Walk does not produce any errors by itself, it just wraps errors returned by the given callback function f.
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

func (e walkError) Unwrap() error {
	return e.err
}

func (e walkError) Cause() error {
	return e.err
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
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
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
