/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package walk_test

import (
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/sap/go-generics/slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/walk"
)

var _ = Describe("testing: walk.go", func() {

	var nodeList []ListNode

	BeforeEach(func() {
		nodeList = nil
	})

	var record walk.WalkFunc = func(x any, path []string, tag reflect.StructTag) error {
		if path != nil && len(path) == 0 {
			return fmt.Errorf("unexpected empty path")
		}

		if slices.Any(nodeList, func(n ListNode) bool {
			return slices.Equal(n.Path, path)
		}) {
			return fmt.Errorf("duplicate path: %v", path)
		}

		if len(path) > 0 && slices.None(nodeList, func(n ListNode) bool {
			return slices.Equal(n.Path, path[:len(path)-1])
		}) {
			return fmt.Errorf("missing parent path: %v", path[:len(path)-1])
		}

		nodeList = append(nodeList, ListNode{
			NodeType: nodeType(x),
			Path:     path,
			Tag:      tag,
		})
		return nil
	}

	It("should walk a string", func() {
		x := new("test")

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.String, Type: "*string"}, Path: nil, Tag: ""},
		}))
	})

	It("should walk an int", func() {
		x := new(123)

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: nil, Tag: ""},
		}))
	})

	It("should walk a float", func() {
		x := new(1.23)

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Float64, Type: "*float64"}, Path: nil, Tag: ""},
		}))
	})

	It("should walk a bool", func() {
		x := new(true)

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Bool, Type: "*bool"}, Path: nil, Tag: ""},
		}))
	})

	It("should walk an empty array", func() {
		x := new([3]int{})

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Array, Type: "*[3]int"}, Path: nil, Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(0), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(1), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(2), Tag: ""},
		}))
	})

	It("should walk a non-empty array", func() {
		x := new([3]int{1, 2, 3})

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Array, Type: "*[3]int"}, Path: nil, Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(0), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(1), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(2), Tag: ""},
		}))
	})

	It("should walk an empty slice", func() {
		x := new([]int{})

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Slice, Type: "*[]int"}, Path: nil, Tag: ""},
		}))
	})

	It("should walk a non-empty slice", func() {
		x := new([]int{1, 2, 3})

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(Equal([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Slice, Type: "*[]int"}, Path: nil, Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(0), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(1), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path(2), Tag: ""},
		}))
	})

	It("should walk a map", func() {
		x := new(map[string]int{"a": 1, "b": 2, "c": 3})

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(ConsistOf([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Map, Type: "*map[string]int"}, Path: nil, Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Int, Type: "int"}, Path: path("a"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Int, Type: "int"}, Path: path("b"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Int, Type: "int"}, Path: path("c"), Tag: ""},
		}))
	})

	It("should walk a zero struct", func() {
		type T struct {
			Int int
		}

		x := &struct {
			String           string `test:"a"`
			Int              int
			Float64          float64
			Bool             bool
			Array            [3]int `test:"b"`
			Slice            []int
			Map              map[string]int
			Struct           T `test:"c"`
			Interface        fmt.Stringer
			unexportedInt    int
			PtrString        *string
			PtrInt           *int
			PtrFloat64       *float64
			PtrBool          *bool
			PtrArray         *[3]int `test:"b"`
			PtrSlice         *[]int
			PtrMap           *map[string]int
			PtrStruct        *T `test:"c"`
			PtrInterface     *fmt.Stringer
			unexportedPtrInt *int
		}{}

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(ConsistOf([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Struct, Type: "*struct { ... }"}, Path: nil, Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.String, Type: "*string"}, Path: path("String"), Tag: `test:"a"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Int"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Float64, Type: "*float64"}, Path: path("Float64"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Bool, Type: "*bool"}, Path: path("Bool"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Array, Type: "*[3]int"}, Path: path("Array"), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Array", 0), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Array", 1), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Array", 2), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Slice, Type: "*[]int"}, Path: path("Slice"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Map, Type: "*map[string]int"}, Path: path("Map"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Struct, Type: "*walk_test.T"}, Path: path("Struct"), Tag: `test:"c"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Struct", "Int"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Interface, Type: "*fmt.Stringer"}, Path: path("Interface"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.String, Type: "*string"}, Path: path("PtrString"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrInt"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Float64, Type: "*float64"}, Path: path("PtrFloat64"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Bool, Type: "*bool"}, Path: path("PtrBool"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Array, Type: "*[3]int"}, Path: path("PtrArray"), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Slice, Type: "*[]int"}, Path: path("PtrSlice"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Map, Type: "*map[string]int"}, Path: path("PtrMap"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Struct, Type: "*walk_test.T"}, Path: path("PtrStruct"), Tag: `test:"c"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Interface, Type: "*fmt.Stringer"}, Path: path("PtrInterface"), Tag: ""},
		}))
	})

	It("should walk a populated struct", func() {
		type T struct {
			Int int
		}

		x := &struct {
			String           string `test:"a"`
			Int              int
			Float64          float64
			Bool             bool
			Array            [3]int `test:"b"`
			Slice            []int
			Map              map[string]int
			Struct           T `test:"c"`
			Interface        fmt.Stringer
			unexportedInt    int
			PtrString        *string
			PtrInt           *int
			PtrFloat64       *float64
			PtrBool          *bool
			PtrArray         *[3]int `test:"b"`
			PtrSlice         *[]int
			PtrMap           *map[string]int
			PtrStruct        *T `test:"c"`
			PtrInterface     *fmt.Stringer
			unexportedPtrInt *int
		}{
			String:           "foo",
			Int:              42,
			Float64:          3.14,
			Bool:             true,
			Array:            [3]int{1, 2, 3},
			Slice:            []int{4, 5, 6},
			Map:              map[string]int{"a": 1, "b": 2},
			Struct:           T{Int: 7},
			Interface:        time.Now(),
			unexportedInt:    99,
			PtrString:        new("foo"),
			PtrInt:           new(42),
			PtrFloat64:       new(3.14),
			PtrBool:          new(true),
			PtrArray:         new([3]int{1, 2, 3}),
			PtrSlice:         new([]int{4, 5, 6}),
			PtrMap:           new(map[string]int{"a": 1, "b": 2}),
			PtrStruct:        new(T{Int: 7}),
			PtrInterface:     new(fmt.Stringer(time.Now())),
			unexportedPtrInt: new(99),
		}

		err := walk.Walk(x, record)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList).To(ConsistOf([]ListNode{
			{NodeType: NodeType{Pointer: true, Kind: reflect.Struct, Type: "*struct { ... }"}, Path: nil, Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.String, Type: "*string"}, Path: path("String"), Tag: `test:"a"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Int"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Float64, Type: "*float64"}, Path: path("Float64"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Bool, Type: "*bool"}, Path: path("Bool"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Array, Type: "*[3]int"}, Path: path("Array"), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Array", 0), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Array", 1), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Array", 2), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Slice, Type: "*[]int"}, Path: path("Slice"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Slice", 0), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Slice", 1), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Slice", 2), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Map, Type: "*map[string]int"}, Path: path("Map"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Int, Type: "int"}, Path: path("Map", "a"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Int, Type: "int"}, Path: path("Map", "b"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Struct, Type: "*walk_test.T"}, Path: path("Struct"), Tag: `test:"c"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("Struct", "Int"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Struct, Type: "time.Time"}, Path: path("Interface"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.String, Type: "*string"}, Path: path("PtrString"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrInt"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Float64, Type: "*float64"}, Path: path("PtrFloat64"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Bool, Type: "*bool"}, Path: path("PtrBool"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Array, Type: "*[3]int"}, Path: path("PtrArray"), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrArray", 0), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrArray", 1), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrArray", 2), Tag: `test:"b"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Slice, Type: "*[]int"}, Path: path("PtrSlice"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrSlice", 0), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrSlice", 1), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrSlice", 2), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Map, Type: "*map[string]int"}, Path: path("PtrMap"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Int, Type: "int"}, Path: path("PtrMap", "a"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Int, Type: "int"}, Path: path("PtrMap", "b"), Tag: ""},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Struct, Type: "*walk_test.T"}, Path: path("PtrStruct"), Tag: `test:"c"`},
			{NodeType: NodeType{Pointer: true, Kind: reflect.Int, Type: "*int"}, Path: path("PtrStruct", "Int"), Tag: ""},
			{NodeType: NodeType{Pointer: false, Kind: reflect.Struct, Type: "time.Time"}, Path: path("PtrInterface"), Tag: ""},
		}))
	})

})

type NodeType struct {
	Pointer bool
	Kind    reflect.Kind
	Type    string
}

func nodeType(x any) NodeType {
	t := reflect.ValueOf(x).Type()
	if t.Kind() == reflect.Pointer {
		return NodeType{Pointer: true, Kind: t.Elem().Kind(), Type: typeString(x)}
	} else {
		return NodeType{Pointer: false, Kind: t.Kind(), Type: typeString(x)}
	}
}

func typeString(x any) string {
	s := fmt.Sprintf("%T", x)
	return regexp.MustCompile(`^(\*?struct\s+\{ )(.*)(\s+\})$`).ReplaceAllString(s, "$1...$3")
}

type ListNode struct {
	NodeType NodeType
	Path     []string
	Tag      reflect.StructTag
}

func path(elems ...any) []string {
	return slices.Collect(elems, func(e any) string {
		return fmt.Sprintf("%v", e)
	})
}
