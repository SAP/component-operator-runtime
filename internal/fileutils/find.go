/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package fileutils

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

const (
	FileTypeRegular uint = 1 << iota
	FileTypeDir
	FileTypeSymlink
	FileTypeNamedPipe
	FileTypeSocket
	FileTypeDevice
	FileTypeCharDevice
	FileTypeIrregular
	FileTypeAny = FileTypeRegular | FileTypeDir | FileTypeSymlink | FileTypeNamedPipe | FileTypeSocket | FileTypeDevice | FileTypeCharDevice | FileTypeIrregular
)

func fileTypeFromMode(mode fs.FileMode) uint {
	fileType := uint(0)
	if mode&fs.ModeType == 0 {
		fileType |= FileTypeRegular
	}
	if mode&fs.ModeDir != 0 {
		fileType |= FileTypeDir
	}
	if mode&fs.ModeSymlink != 0 {
		fileType |= FileTypeSymlink
	}
	if mode&fs.ModeNamedPipe != 0 {
		fileType |= FileTypeNamedPipe
	}
	if mode&fs.ModeSocket != 0 {
		fileType |= FileTypeSocket
	}
	if mode&fs.ModeDevice != 0 {
		fileType |= FileTypeDevice
	}
	if mode&fs.ModeCharDevice != 0 {
		fileType |= FileTypeCharDevice
	}
	if mode&fs.ModeIrregular != 0 {
		fileType |= FileTypeIrregular
	}
	return fileType
}

// Search fsys for all files under dir matching namePattern and fileType.
// Resulting paths will be always relative to fsys (cleaned, with no leading dot).
// The parameter dir must not contain any dot or double dot, unless it equals '.' in which case the whole fsys will be searched.
// As an alternative, dir can be empty (which is equivalent to dir == '.').
// Parameters namePattern and fileType may be optionally set to filter the result; namePattern must be a valid file pattern, not
// containing any slashes (otherwise a panic will be raised); the pattern will be matched using path.Match(); an empty namePattern
// will match anything. The parameter fileType may be a (logically or'ed) combination of the constants defined
// in this file; passing any other values will lead to a panic; supplying fileType as zero is the same as passing fileTypeAny.
// The parameter maxDepth can be any integer between 0 and 10000 (where 0 is interpreted as 10000).
// The returned paths will be relative (to the provided fsys), and filepath.Clean() will be run on each entry.
func Find(fsys fs.FS, dir string, namePattern string, fileType uint, maxDepth uint) ([]string, error) {
	if dir == "" {
		dir = "."
	}
	if namePattern == "" {
		namePattern = "*"
	} else if strings.Contains(namePattern, "/") {
		panic("invalid name pattern; must not contain slashes")
	}
	if fileType == 0 {
		fileType = FileTypeAny
	} else if fileType&FileTypeAny != fileType {
		panic("invalid file type")
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
		entryPath := filepath.Clean(dir + "/" + entryName)
		match, err := path.Match(namePattern, entryName)
		if err != nil {
			return nil, err
		}
		if match && (fileTypeFromMode(entryType)&fileType != 0) {
			result = append(result, entryPath)
		}
		if entry.IsDir() && maxDepth > 1 {
			entryResult, err := Find(fsys, entryPath, namePattern, fileType, maxDepth-1)
			if err != nil {
				return nil, err
			}
			result = append(result, entryResult...)
		}
	}

	return result, nil
}
