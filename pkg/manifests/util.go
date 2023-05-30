/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manifests

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
)

// Deep-merge two maps with the usual logic and return the result.
// The first map (x) must be deeply JSON (i.e. consist deeply of JSON values only).
// The maps given as input will not be changed.
func MergeMaps(x, y map[string]any) map[string]any {
	if x == nil {
		x = make(map[string]any)
	} else {
		x = runtime.DeepCopyJSON(x)
	}
	for k, w := range y {
		if v, ok := x[k]; ok {
			if v, ok := v.(map[string]any); ok {
				if _w, ok := w.(map[string]any); ok {
					x[k] = MergeMaps(v, _w)
				} else {
					x[k] = w
				}
			} else {
				x[k] = w
			}
		} else {
			x[k] = w
		}
	}
	return x
}

const (
	fileTypeRegular uint = 1 << iota
	fileTypeDir
	fileTypeSymlink
	fileTypeNamedPipe
	fileTypeSocket
	fileTypeDevice
	fileTypeCharDevice
	fileTypeIrregular
	fileTypeAny = fileTypeRegular | fileTypeDir | fileTypeSymlink | fileTypeNamedPipe | fileTypeSocket | fileTypeDevice | fileTypeCharDevice | fileTypeIrregular
)

func fileTypeFromMode(mode fs.FileMode) uint {
	fileType := uint(0)
	if mode&fs.ModeType == 0 {
		fileType |= fileTypeRegular
	}
	if mode&fs.ModeDir != 0 {
		fileType |= fileTypeDir
	}
	if mode&fs.ModeSymlink != 0 {
		fileType |= fileTypeSymlink
	}
	if mode&fs.ModeNamedPipe != 0 {
		fileType |= fileTypeNamedPipe
	}
	if mode&fs.ModeSocket != 0 {
		fileType |= fileTypeSocket
	}
	if mode&fs.ModeDevice != 0 {
		fileType |= fileTypeDevice
	}
	if mode&fs.ModeCharDevice != 0 {
		fileType |= fileTypeCharDevice
	}
	if mode&fs.ModeIrregular != 0 {
		fileType |= fileTypeIrregular
	}
	return fileType
}

func find(fsys fs.FS, dir string, namePattern string, fileType uint, maxDepth uint) ([]string, error) {
	if strings.Contains(namePattern, "/") {
		return nil, fmt.Errorf("invalid name pattern; must not contain slashes")
	}
	if fileType == 0 {
		fileType = fileTypeAny
	} else if fileType&fileTypeAny != fileType {
		return nil, fmt.Errorf("invalid file type")
	}
	if maxDepth == 0 {
		maxDepth = 10000
	} else if maxDepth > 10000 {
		// for security; never descend infinitely
		return nil, fmt.Errorf("invalid maximum depth; must not exceed 10000")
	}

	var result []string

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		// TODO: is it ok to tolerate non-existing dir, or should we (optionally) fail here ?
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	for _, entry := range entries {
		entryName := entry.Name()
		entryType := entry.Type()
		match, err := path.Match(namePattern, entryName)
		if err != nil {
			return nil, err
		}
		if match && (fileTypeFromMode(entryType)&fileType != 0) {
			result = append(result, dir+"/"+entryName)
		}
		if entry.IsDir() && maxDepth > 1 {
			entryResult, err := find(fsys, dir+"/"+entryName, namePattern, fileType, maxDepth-1)
			if err != nil {
				return nil, err
			}
			result = append(result, entryResult...)
		}
	}

	return result, nil
}
