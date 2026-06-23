/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package fileutils_test

import (
	"fmt"
	"io/fs"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/fileutils"
)

var _ = Describe("testing: find.go", func() {

	var fsys fs.FS

	BeforeEach(func() {
		fsys = os.DirFS("testdata/fs")
	})

	It("should find everything", func() {
		files, err := fileutils.Find(fsys, "", "", 0, 0)
		fmt.Println(files)
		Expect(err).NotTo(HaveOccurred())
		Expect(files).NotTo(BeEmpty())
	})

})
