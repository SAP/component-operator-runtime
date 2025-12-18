/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"fmt"
	"strconv"
	"strings"

	legacyerrors "github.com/pkg/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	annotationKeyHook             = "helm.sh/hook"
	annotationKeyHookWeight       = "helm.sh/hook-weight"
	annotationKeyHookDeletePolicy = "helm.sh/hook-delete-policy"
)

// parse helm hook properties from object, return nil if annotation helm.sh/hook is not set
func ParseHookMetadata(object client.Object) (*HookMetadata, error) {
	metadata := &HookMetadata{}
	annotations := object.GetAnnotations()

	if value, ok := annotations[annotationKeyHook]; ok {
		metadata.Types = strings.Split(value, ",")
		for _, t := range metadata.Types {
			switch t {
			case HookTypePreInstall, HookTypePostInstall, HookTypePreUpgrade, HookTypePostUpgrade,
				HookTypePreDelete, HookTypePostDelete, HookTypePreRollback, HookTypePostRollback,
				HookTypeTest, HookTypeTestSuccess:
			default:
				return nil, fmt.Errorf("invalid hook type: %s", t)
			}
		}
	} else {
		return nil, nil
	}

	if value, ok := annotations[annotationKeyHookWeight]; ok {
		weight, err := strconv.Atoi(value)
		if err != nil {
			return nil, legacyerrors.Wrapf(err, "invalid hook weight: %s", value)
		}
		if weight < HookMinWeight || weight > HookMaxWeight {
			return nil, fmt.Errorf("invalid hook weight: %d (allowed range: %d..%d)", weight, HookMinWeight, HookMaxWeight)
		}
		metadata.Weight = weight
	} else {
		metadata.Weight = 0
	}

	if value, ok := annotations[annotationKeyHookDeletePolicy]; ok {
		metadata.DeletePolicies = strings.Split(value, ",")
		for _, p := range metadata.DeletePolicies {
			switch p {
			case HookDeletePolicyBeforeHookCreation, HookDeletePolicyHookSucceeded, HookDeletePolicyHookFailed:
			default:
				return nil, fmt.Errorf("invalid hook deletion policy: %s", p)
			}
		}
	} else {
		metadata.DeletePolicies = []string{HookDeletePolicyBeforeHookCreation}
	}

	return metadata, nil
}
