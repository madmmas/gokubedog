/*
Copyright 2025.

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

package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// kindToGVR maps Kubernetes resource kinds to their GroupVersionResource
var kindToGVR = map[string]schema.GroupVersionResource{
	// Core resources
	"Pod": {
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	},
	"Service": {
		Group:    "",
		Version:  "v1",
		Resource: "services",
	},
	"ConfigMap": {
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	},
	"Secret": {
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	},
	"Namespace": {
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	},
	"PersistentVolume": {
		Group:    "",
		Version:  "v1",
		Resource: "persistentvolumes",
	},
	"PersistentVolumeClaim": {
		Group:    "",
		Version:  "v1",
		Resource: "persistentvolumeclaims",
	},

	// Apps resources
	"Deployment": {
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	},
	"ReplicaSet": {
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	},
	"StatefulSet": {
		Group:    "apps",
		Version:  "v1",
		Resource: "statefulsets",
	},
	"DaemonSet": {
		Group:    "apps",
		Version:  "v1",
		Resource: "daemonsets",
	},

	// Networking resources
	"NetworkPolicy": {
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "networkpolicies",
	},
	"Ingress": {
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	},

	// RBAC resources
	"Role": {
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "roles",
	},
	"RoleBinding": {
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "rolebindings",
	},
	"ClusterRole": {
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterroles",
	},
	"ClusterRoleBinding": {
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Resource: "clusterrolebindings",
	},

	// Storage resources
	"StorageClass": {
		Group:    "storage.k8s.io",
		Version:  "v1",
		Resource: "storageclasses",
	},

	// Policy resources
	"PodDisruptionBudget": {
		Group:    "policy",
		Version:  "v1",
		Resource: "poddisruptionbudgets",
	},

	// Batch resources
	"Job": {
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	},
	"CronJob": {
		Group:    "batch",
		Version:  "v1",
		Resource: "cronjobs",
	},
}

// GetGVRFromKind dynamically maps a kind to its GroupVersionResource
func GetGVRFromKind(kind string) (schema.GroupVersionResource, error) {
	if gvr, exists := kindToGVR[kind]; exists {
		return gvr, nil
	}
	return schema.GroupVersionResource{}, fmt.Errorf("unsupported kind: %s", kind)
}

// GetSupportedKinds returns a list of all supported resource kinds
func GetSupportedKinds() []string {
	kinds := make([]string, 0, len(kindToGVR))
	for kind := range kindToGVR {
		kinds = append(kinds, kind)
	}
	return kinds
}

// IsKindSupported checks if a given kind is supported
func IsKindSupported(kind string) bool {
	_, exists := kindToGVR[kind]
	return exists
}
