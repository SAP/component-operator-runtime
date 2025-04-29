/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"fmt"
	"strings"
)

/*
func getString(data map[string]any, key string) (string, bool, bool) {
	if v, ok := data[key]; ok {
		if v, ok := v.(string); ok {
			return v, true, true
		}
		return "", true, false
	}
	return "", false, false
}
*/

/*
func getInt(data map[string]any, key string) (int64, bool, bool) {
	if v, ok := data[key]; ok {
		if v, ok := v.(int64); ok {
			return v, true, true
		}
		return 0, true, false
	}
	return 0, false, false
}
*/

/*
func getFloat(data map[string]any, key string) (float64, bool, bool) {
	if v, ok := data[key]; ok {
		if v, ok := v.(float64); ok {
			return v, true, true
		}
		return 0, true, false
	}
	return 0, false, false
}
*/

/*
func getBool(data map[string]any, key string) (bool, bool, bool) {
	if v, ok := data[key]; ok {
		if v, ok := v.(bool); ok {
			return v, true, true
		}
		return false, true, false
	}
	return false, false, false
}
*/

/*
func getArray(data map[string]any, key string) ([]any, bool, bool) {
	if v, ok := data[key]; ok {
		if v, ok := v.([]any); ok {
			return v, true, true
		}
		return nil, true, false
	}
	return nil, false, false
}
*/

func must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}

func getMap(data map[string]any, key string) (map[string]any, bool, bool) {
	if v, ok := data[key]; ok {
		if v, ok := v.(map[string]any); ok {
			return v, true, true
		}
		return nil, true, false
	}
	return nil, false, false
}

func dig(data map[string]any, paths ...string) (any, bool) {
	var keys []string
	for _, path := range paths {
		keys = append(keys, splitPath(path)...)
	}
	if len(keys) == 0 {
		return data, true
	}
	for i, key := range keys {
		value, ok := data[key]
		if !ok {
			return nil, false
		}
		if i == len(keys)-1 {
			return value, true
		}
		data, ok = value.(map[string]any)
		if !ok {
			return nil, false
		}
	}
	return nil, false
}

/*
func digString(data map[string]any, paths ...string) (string, bool, bool) {
	if v, ok := dig(data, paths...); ok {
		if v, ok := v.(string); ok {
			return v, true, true
		}
		return "", true, false
	}
	return "", false, false
}
*/

/*
func digInt(data map[string]any, paths ...string) (int64, bool, bool) {
	if v, ok := dig(data, paths...); ok {
		if v, ok := v.(int64); ok {
			return v, true, true
		}
		return 0, true, false
	}
	return 0, false, false
}
*/

/*
func digFloat(data map[string]any, paths ...string) (float64, bool, bool) {
	if v, ok := dig(data, paths...); ok {
		if v, ok := v.(float64); ok {
			return v, true, true
		}
		return 0, true, false
	}
	return 0, false, false
}
*/

func digBool(data map[string]any, paths ...string) (bool, bool, bool) {
	if v, ok := dig(data, paths...); ok {
		if v, ok := v.(bool); ok {
			return v, true, true
		}
		return false, true, false
	}
	return false, false, false
}

/*
func digArray(data map[string]any, paths ...string) ([]any, bool, bool) {
	if v, ok := dig(data, paths...); ok {
		if v, ok := v.([]any); ok {
			return v, true, true
		}
		return nil, true, false
	}
	return nil, false, false
}
*/

func digMap(data map[string]any, paths ...string) (map[string]any, bool, bool) {
	if v, ok := dig(data, paths...); ok {
		if v, ok := v.(map[string]any); ok {
			return v, true, true
		}
		return nil, true, false
	}
	return nil, false, false
}

func undig(data map[string]any, value any, paths ...string) error {
	var keys []string
	for _, path := range paths {
		keys = append(keys, splitPath(path)...)
	}
	if len(keys) == 0 {
		panic("cannot undig into an empty path")
	}
	for _, key := range keys[0 : len(keys)-1] {
		if value, ok := data[key]; ok {
			if _, ok := value.(map[string]any); !ok {
				return fmt.Errorf("cannot undig into fields which are not a string-keyed map")
			}
		} else {
			data[key] = make(map[string]any)
		}
		data = data[key].(map[string]any)
	}
	data[keys[len(keys)-1]] = value
	return nil
}

func splitPath(s string) []string {
	// TODO: allow dots in s to be escaped by backslash
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}
