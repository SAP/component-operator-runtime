/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package fileutils_test

import (
	"io/fs"
	"os"
	"strings"

	"github.com/sap/go-generics/slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/fileutils"
)

var _ = Describe("testing: find.go", func() {

	var fsys fs.FS
	var directories []string
	var files []string
	var symlinks []string

	BeforeEach(func() {
		fsys = os.DirFS("testdata/fs")
		directories = []string{
			"d",
			"d/d",
			"e",
			"e/e",
			"e/e/e",
		}
		files = []string{
			"a.x",
			"d/a.y",
			"d/b.y",
			"d/d/c.x",
			"d/d/a.y",
			"d/d/a.x",
			"d/d/b.x",
			"e/e/e/c.y",
		}
		symlinks = []string{
			"d/la.y",
			"d/ld",
		}
	})

	It("should find everything", func() {
		paths, err := fileutils.Find(fsys, "", "", 0, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(concat(directories, files, symlinks)))

		paths, err = fileutils.Find(fsys, ".", "*", fileutils.FileTypeAny, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(concat(directories, files, symlinks)))
	})

	It("should find directories", func() {
		paths, err := fileutils.Find(fsys, "", "", fileutils.FileTypeDir, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(concat(directories)))
	})

	It("should find files", func() {
		paths, err := fileutils.Find(fsys, "", "", fileutils.FileTypeRegular, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(concat(files)))
	})

	It("should find symlinks", func() {
		paths, err := fileutils.Find(fsys, "", "", fileutils.FileTypeSymlink, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(concat(symlinks)))
	})

	It("should find files or symlinks", func() {
		paths, err := fileutils.Find(fsys, "", "", fileutils.FileTypeRegular|fileutils.FileTypeSymlink, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(concat(files, symlinks)))
	})

	It("should find everything in a subdirectory", func() {
		paths, err := fileutils.Find(fsys, "d", "", 0, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(havingPrefix(concat(directories, files, symlinks), "d/")))
	})

	It("should honor namePattern", func() {
		paths, err := fileutils.Find(fsys, "", "*.x", fileutils.FileTypeRegular, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(havingSuffix(files, ".x")))
	})

	It("should honor maxDepth", func() {
		paths, err := fileutils.Find(fsys, "", "", fileutils.FileTypeRegular, 1)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(havingMaxDepth(files, 1)))
	})

	It("should not follow symlinks", func() {
		paths, err := fileutils.Find(fsys, "", "", fileutils.FileTypeAny, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(havingPrefix(paths, "d/ld/")).To(BeEmpty())
	})

	It("should allow dir to be a symlink", func() {
		paths, err := fileutils.Find(fsys, "d/ld", "", fileutils.FileTypeAny, 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(paths).To(ConsistOf(havingPrefix(replacingPrefix(concat(directories, files, symlinks), "d/d/", "d/ld/"), "d/ld/")))
	})

})

func concat[T any](slices ...[]T) []T {
	var result []T
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

func havingPrefix(paths []string, prefix string) []string {
	return slices.Select(paths, func(path string) bool {
		return strings.HasPrefix(path, prefix)
	})
}

func havingSuffix(paths []string, suffix string) []string {
	return slices.Select(paths, func(path string) bool {
		return strings.HasSuffix(path, suffix)
	})
}

func havingMaxDepth(paths []string, maxDepth int) []string {
	return slices.Select(paths, func(path string) bool {
		return strings.Count(path, "/") < maxDepth
	})
}

func replacingPrefix(paths []string, prefix, replacement string) []string {
	return slices.Collect(paths, func(path string) string {
		if strings.HasPrefix(path, prefix) {
			return replacement + path[len(prefix):]
		}
		return path
	})
}
