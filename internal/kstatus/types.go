/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kstatus

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

// TODO: the StatusAnalyzer interface should be public.

// The StatusAnalyzer interface models types which allow to extract a kstatus-compatible status from an object.
type StatusAnalyzer interface {
	ComputeStatus(object *unstructured.Unstructured) (Status, error)
}

type Status kstatus.Status

const (
	InProgressStatus  Status = Status(kstatus.InProgressStatus)
	FailedStatus      Status = Status(kstatus.FailedStatus)
	CurrentStatus     Status = Status(kstatus.CurrentStatus)
	TerminatingStatus Status = Status(kstatus.TerminatingStatus)
	NotFoundStatus    Status = Status(kstatus.NotFoundStatus)
	UnknownStatus     Status = Status(kstatus.UnknownStatus)
)
