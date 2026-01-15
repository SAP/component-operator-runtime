/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/pkg/manifests"
)

var _ = Describe("testing: util.go", func() {
	DescribeTable("testing: MergeMaps()",
		func(xs, ys, rs string) {
			x := jsonUnmarshal(xs)
			y := jsonUnmarshal(ys)
			Expect(manifests.MergeMaps(x, y)).To(Equal(jsonUnmarshal(rs)))
			Expect(x).To(Equal(jsonUnmarshal(xs)))
			Expect(y).To(Equal(jsonUnmarshal(ys)))
		},

		Entry(nil,
			`{
			"a": {
				"a": {
					"x": 1
				},
				"b": {
					"x": 1
				},
				"u": [
					1
				],
				"v": [
					1
				],
				"x": 1,
				"y": 1
			},
			"b": {
				"x": 1
			},
			"u": [
				1
			],
			"v": [
				1
			],
			"x": 1,
			"y": 1
		}`,
			`{
			"a": {
				"a": {
					"x": 2,
					"z": 2
				},
				"u": 2,
				"x": 2,
				"z": 2
			},
			"u": 2,
			"x": 2,
			"z": 2
		}`,
			`{
			"a": {
				"a": {
					"x": 2,
					"z": 2
				},
				"b": {
					"x": 1
				},
				"u": 2,
				"v": [
					1
				],
				"x": 2,
				"y": 1,
				"z": 2
			},
			"b": {
				"x": 1
			},
			"u": 2,
			"v": [
				1
			],
			"x": 2,
			"y": 1,
			"z": 2
		}`,
		),
	)

	DescribeTable("testing: MergeMapInto()",
		func(xs, ys, rs string) {
			x := jsonUnmarshal(xs)
			y := jsonUnmarshal(ys)
			manifests.MergeMapInto(x, y)
			Expect(x).To(Equal(jsonUnmarshal(rs)))
			Expect(y).To(Equal(jsonUnmarshal(ys)))
		},

		Entry(nil,
			`{
			"a": {
				"a": {
					"x": 1
				},
				"b": {
					"x": 1
				},
				"u": [
					1
				],
				"v": [
					1
				],
				"x": 1,
				"y": 1
			},
			"b": {
				"x": 1
			},
			"u": [
				1
			],
			"v": [
				1
			],
			"x": 1,
			"y": 1
		}`,
			`{
			"a": {
				"a": {
					"x": 2,
					"z": 2
				},
				"u": 2,
				"x": 2,
				"z": 2
			},
			"u": 2,
			"x": 2,
			"z": 2
		}`,
			`{
			"a": {
				"a": {
					"x": 2,
					"z": 2
				},
				"b": {
					"x": 1
				},
				"u": 2,
				"v": [
					1
				],
				"x": 2,
				"y": 1,
				"z": 2
			},
			"b": {
				"x": 1
			},
			"u": 2,
			"v": [
				1
			],
			"x": 2,
			"y": 1,
			"z": 2
		}`,
		),
	)
})

func jsonUnmarshal(s string) (x map[string]any) {
	if err := json.Unmarshal([]byte(s), &x); err != nil {
		panic(err)
	}
	return
}
