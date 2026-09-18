/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types_test

import (
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/pkg/types"
)

var _ = Describe("testing: errors.go", func() {

	It("should return a valid RetriableError", func() {
		err := CustomError{Message: "test error"}
		after := 5 * time.Second

		rerr := types.NewRetriableError(err, &after)
		Expect(rerr.Error()).To(Equal("test error"))
		Expect(rerr.Unwrap()).To(Equal(err))
		Expect(rerr.Cause()).To(Equal(err))
		Expect(rerr.RetryAfter()).To(Equal(&after))
	})

	It("should work correctly with the errors package", func() {
		innerError := CustomError{Message: "inner error"}
		retriableError := types.NewRetriableError(innerError, new(5*time.Second))
		outerError := fmt.Errorf("outer error: %w", retriableError)

		var unwrappedRetriableError types.RetriableError
		Expect(errors.As(outerError, &unwrappedRetriableError)).To(BeTrue())
		Expect(unwrappedRetriableError).To(BeIdenticalTo(retriableError))
		Expect(errors.Is(unwrappedRetriableError, innerError)).To(BeTrue())

		unwrappedRetriableError, ok := errors.AsType[types.RetriableError](outerError)
		Expect(ok).To(BeTrue())
		Expect(unwrappedRetriableError).To(BeIdenticalTo(retriableError))
		Expect(errors.Is(unwrappedRetriableError, innerError)).To(BeTrue())

		Expect(errors.Is(outerError, unwrappedRetriableError)).To(BeTrue())
	})

})

type CustomError struct {
	Message string
}

func (e CustomError) Error() string {
	return e.Message
}
