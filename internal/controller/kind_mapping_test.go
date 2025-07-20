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
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetGVRFromKind(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		wantGVR schema.GroupVersionResource
		wantErr bool
	}{
		{
			name: "NetworkPolicy should map correctly",
			kind: "NetworkPolicy",
			wantGVR: schema.GroupVersionResource{
				Group:    "networking.k8s.io",
				Version:  "v1",
				Resource: "networkpolicies",
			},
			wantErr: false,
		},
		{
			name: "Pod should map correctly",
			kind: "Pod",
			wantGVR: schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			wantErr: false,
		},
		{
			name: "Deployment should map correctly",
			kind: "Deployment",
			wantGVR: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			wantErr: false,
		},
		{
			name:    "UnknownKind should return error",
			kind:    "UnknownKind",
			wantGVR: schema.GroupVersionResource{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGVR, err := GetGVRFromKind(tt.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGVRFromKind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gotGVR != tt.wantGVR {
				t.Errorf("GetGVRFromKind() = %v, want %v", gotGVR, tt.wantGVR)
			}
		})
	}
}

func TestGetSupportedKinds(t *testing.T) {
	kinds := GetSupportedKinds()

	// Check that we have a reasonable number of supported kinds
	if len(kinds) < 10 {
		t.Errorf("Expected at least 10 supported kinds, got %d", len(kinds))
	}

	// Check that some expected kinds are present
	expectedKinds := []string{"Pod", "Deployment", "NetworkPolicy", "Service"}
	for _, expectedKind := range expectedKinds {
		found := false
		for _, kind := range kinds {
			if kind == expectedKind {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected kind %s not found in supported kinds", expectedKind)
		}
	}
}

func TestIsKindSupported(t *testing.T) {
	tests := []struct {
		name           string
		kind           string
		expectedResult bool
	}{
		{
			name:           "Pod should be supported",
			kind:           "Pod",
			expectedResult: true,
		},
		{
			name:           "Deployment should be supported",
			kind:           "Deployment",
			expectedResult: true,
		},
		{
			name:           "NetworkPolicy should be supported",
			kind:           "NetworkPolicy",
			expectedResult: true,
		},
		{
			name:           "UnknownKind should not be supported",
			kind:           "UnknownKind",
			expectedResult: false,
		},
		{
			name:           "Empty string should not be supported",
			kind:           "",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsKindSupported(tt.kind)
			if result != tt.expectedResult {
				t.Errorf("IsKindSupported(%s) = %v, want %v", tt.kind, result, tt.expectedResult)
			}
		})
	}
}
