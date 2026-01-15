/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"bytes"
	"fmt"
	"strings"

	legacyerrors "github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
)

// TODO: needs refactoring

func replaceFieldManager(managedFields []metav1.ManagedFieldsEntry, managerPrefixes []string, manager string) ([]metav1.ManagedFieldsEntry, bool, error) {
	var managerEntry metav1.ManagedFieldsEntry
	empty := metav1.ManagedFieldsEntry{}

	for _, entry := range managedFields {
		if entry.Manager == manager && entry.Operation == metav1.ManagedFieldsOperationApply {
			managerEntry = entry
		}
	}

	entries := make([]metav1.ManagedFieldsEntry, 0, len(managedFields))
	changed := false

	for _, entry := range managedFields {
		if entry == managerEntry {
			continue
		}
		if entry.Subresource != "" {
			entries = append(entries, entry)
			continue
		}
		if entry.Manager != manager && !slices.Any(managerPrefixes, func(s string) bool { return strings.HasPrefix(entry.Manager, s) }) {
			entries = append(entries, entry)
			continue
		}
		if managerEntry == empty {
			entry.Manager = manager
			entry.Operation = metav1.ManagedFieldsOperationApply
			managerEntry = entry
			changed = true
			continue
		}
		mergedFields, err := mergeManagedFieldsV1(managerEntry.FieldsV1, entry.FieldsV1)
		if err != nil {
			return nil, false, legacyerrors.Wrap(err, "unable to merge managed fields")
		}
		managerEntry.FieldsV1 = mergedFields
		changed = true
	}

	return append(entries, managerEntry), changed, nil
}

func mergeManagedFieldsV1(prevField *metav1.FieldsV1, newField *metav1.FieldsV1) (*metav1.FieldsV1, error) {
	if prevField == nil && newField == nil {
		return nil, nil
	}

	if prevField == nil {
		return newField, nil
	}

	if newField == nil {
		return prevField, nil
	}

	prevSet, err := fieldsToSet(*prevField)
	if err != nil {
		return nil, err
	}

	newSet, err := fieldsToSet(*newField)
	if err != nil {
		return nil, err
	}

	unionSet := prevSet.Union(&newSet)
	mergedField, err := setToFields(*unionSet)
	if err != nil {
		return nil, fmt.Errorf("unable to convert managed set to field: %s", err)
	}

	return &mergedField, nil
}

func fieldsToSet(f metav1.FieldsV1) (s fieldpath.Set, err error) {
	err = s.FromJSON(bytes.NewReader(f.Raw))
	return s, err
}

func setToFields(s fieldpath.Set) (f metav1.FieldsV1, err error) {
	f.Raw, err = s.ToJSON()
	return f, err
}
