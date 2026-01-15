/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package release

import (
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/reconciler"
)

const (
	dataKeyVersion           = "version"
	dataKeyCreationTimestamp = "creationTimestamp"
	dataKeyUpdateTimestamp   = "updateTimestamp"
	dataKeyRevision          = "revision"
	dataKeyInventory         = "inventory"
	dataKeyState             = "state"
)

type Release struct {
	namespace         string
	name              string
	creationTimestamp *time.Time
	updateTimestamp   *time.Time
	configMap         *corev1.ConfigMap
	Revision          int64
	Inventory         []*reconciler.InventoryItem
	State             component.State
}

func (r *Release) GetNamespace() string {
	return r.namespace
}

func (r *Release) GetName() string {
	return r.name
}

func (r *Release) IsDeleting() bool {
	return !r.configMap.DeletionTimestamp.IsZero()
}

func (r *Release) GetCreationTimestamp() *time.Time {
	return r.creationTimestamp
}

func (r *Release) GetUpdateTimestamp() *time.Time {
	return r.updateTimestamp
}

func (r *Release) importData() error {
	if creationTimestampData, ok := r.configMap.Data[dataKeyCreationTimestamp]; ok {
		creationTimestamp, err := time.Parse(time.RFC3339, creationTimestampData)
		if err != nil {
			return err
		}
		r.creationTimestamp = &creationTimestamp
	} else {
		r.creationTimestamp = nil
	}

	if updateTimestampData, ok := r.configMap.Data[dataKeyUpdateTimestamp]; ok {
		updateTimestamp, err := time.Parse(time.RFC3339, updateTimestampData)
		if err != nil {
			return err
		}
		r.updateTimestamp = &updateTimestamp
	} else {
		r.updateTimestamp = nil
	}

	if revisionData, ok := r.configMap.Data[dataKeyRevision]; ok {
		revision, err := strconv.ParseInt(revisionData, 10, 64)
		if err != nil {
			return err
		}
		r.Revision = revision
	} else {
		r.Revision = 0
	}

	if inventoryData, ok := r.configMap.Data[dataKeyInventory]; ok {
		if err := kyaml.Unmarshal([]byte(inventoryData), &r.Inventory); err != nil {
			return err
		}
	} else {
		r.Inventory = nil
	}

	if stateData, ok := r.configMap.Data[dataKeyState]; ok {
		r.State = component.State(stateData)
	} else {
		r.State = ""
	}

	return nil
}

func (r *Release) exportData() error {
	if r.configMap.Data == nil {
		r.configMap.Data = make(map[string]string)
	}

	r.configMap.Data[dataKeyVersion] = "1"

	if r.creationTimestamp != nil {
		r.configMap.Data[dataKeyCreationTimestamp] = r.creationTimestamp.Format(time.RFC3339)
	} else {
		delete(r.configMap.Data, dataKeyCreationTimestamp)
	}

	if r.updateTimestamp != nil {
		r.configMap.Data[dataKeyUpdateTimestamp] = r.updateTimestamp.Format(time.RFC3339)
	} else {
		delete(r.configMap.Data, dataKeyUpdateTimestamp)
	}

	r.configMap.Data[dataKeyRevision] = strconv.FormatInt(r.Revision, 10)

	if len(r.Inventory) > 0 {
		inventoryRawData, err := kyaml.Marshal(r.Inventory)
		if err != nil {
			return err
		}
		r.configMap.Data[dataKeyInventory] = string(inventoryRawData)
	} else {
		delete(r.configMap.Data, dataKeyInventory)
	}

	if r.State != "" {
		r.configMap.Data[dataKeyState] = string(r.State)
	} else {
		delete(r.configMap.Data, dataKeyState)
	}

	return nil
}
