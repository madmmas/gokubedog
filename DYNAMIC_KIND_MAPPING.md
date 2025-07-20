# Dynamic Kind Mapping in PolicyProfile Controller

This document explains how the PolicyProfile controller dynamically maps Kubernetes resource kinds to their GroupVersionResource (GVR) for policy enforcement.

## Overview

The PolicyProfile controller supports dynamic mapping from resource kinds (like `Pod`, `Deployment`, `NetworkPolicy`) to their corresponding GroupVersionResource, allowing you to apply policies to any supported Kubernetes resource type.

## File Structure

The kind mapping functionality is organized into separate files for better maintainability:

- `internal/controller/kind_mapping.go` - Contains the static mapping and utility functions
- `internal/controller/policyprofile_controller.go` - Main controller logic with dynamic discovery
- `internal/controller/kind_mapping_test.go` - Tests for the kind mapping functions

## How It Works

### 1. Static Mapping (Primary Method)

The controller includes a comprehensive static mapping of common Kubernetes resource kinds to their GVRs in `kind_mapping.go`:

```go
var kindToGVR = map[string]schema.GroupVersionResource{
    "Pod": {
        Group:    "",
        Version:  "v1",
        Resource: "pods",
    },
    "Deployment": {
        Group:    "apps",
        Version:  "v1",
        Resource: "deployments",
    },
    "NetworkPolicy": {
        Group:    "networking.k8s.io",
        Version:  "v1",
        Resource: "networkpolicies",
    },
    // ... more mappings
}
```

### 2. Dynamic Discovery (Fallback Method)

For resource kinds not in the static mapping, the controller can dynamically discover them using the Kubernetes API discovery mechanism:

1. Queries all available API groups
2. Searches through each group version for the specified kind
3. Caches the result for future use
4. Returns the appropriate GVR

## API Functions

The `kind_mapping.go` file provides several utility functions:

### `GetGVRFromKind(kind string) (schema.GroupVersionResource, error)`
Maps a kind to its GroupVersionResource using the static mapping.

### `GetSupportedKinds() []string`
Returns a list of all supported resource kinds.

### `IsKindSupported(kind string) bool`
Checks if a given kind is supported.

## Supported Resource Types

### Core Resources (v1)
- `Pod`
- `Service`
- `ConfigMap`
- `Secret`
- `Namespace`
- `PersistentVolume`
- `PersistentVolumeClaim`

### Apps Resources (apps/v1)
- `Deployment`
- `ReplicaSet`
- `StatefulSet`
- `DaemonSet`

### Networking Resources (networking.k8s.io/v1)
- `NetworkPolicy`
- `Ingress`

### RBAC Resources (rbac.authorization.k8s.io/v1)
- `Role`
- `RoleBinding`
- `ClusterRole`
- `ClusterRoleBinding`

### Storage Resources (storage.k8s.io/v1)
- `StorageClass`

### Policy Resources (policy/v1)
- `PodDisruptionBudget`

### Batch Resources (batch/v1)
- `Job`
- `CronJob`

## Usage Examples

### Example 1: Network Policy Enforcement

```yaml
apiVersion: watchdog.bizaikube.io/v1alpha1
kind: PolicyProfile
metadata:
  name: restrict-ingress
spec:
  match:
    kind: NetworkPolicy
    namespace: prod-*
  policy:
    denyIngress: "true"
    labels.security: "strict"
```

### Example 2: Pod Security Policy

```yaml
apiVersion: watchdog.bizaikube.io/v1alpha1
kind: PolicyProfile
metadata:
  name: pod-security-policy
spec:
  match:
    kind: Pod
    namespace: default
  policy:
    security.kubernetes.io/psp: "restricted"
    app.kubernetes.io/managed-by: "policy-controller"
```

### Example 3: Deployment Security Policy

```yaml
apiVersion: watchdog.bizaikube.io/v1alpha1
kind: PolicyProfile
metadata:
  name: deployment-security-policy
spec:
  match:
    kind: Deployment
    namespace: prod-*
  policy:
    security.kubernetes.io/run-as-non-root: "true"
    security.kubernetes.io/read-only-root-filesystem: "true"
    app.kubernetes.io/environment: "production"
```

## Adding New Resource Types

### Method 1: Add to Static Mapping

To add support for a new resource type, add it to the `kindToGVR` map in `internal/controller/kind_mapping.go`:

```go
"NewResource": {
    Group:    "new.api.group",
    Version:  "v1",
    Resource: "newresources",
},
```

### Method 2: Dynamic Discovery

For custom resources or resources not in the static mapping, the controller will automatically discover them at runtime using the Kubernetes API discovery mechanism.

## Error Handling

- If a kind is not found in the static mapping and cannot be discovered dynamically, the controller will log an error and skip processing that PolicyProfile
- The controller will continue processing other PolicyProfiles even if one fails

## Performance Considerations

- Static mapping is used first for better performance
- Dynamic discovery results are cached to avoid repeated API calls
- The discovery cache is maintained per controller instance

## Testing

You can test the kind mapping functionality using the provided tests:

```bash
# Test the kind mapping functions
go test ./internal/controller -v -run TestGetGVRFromKind
go test ./internal/controller -v -run TestGetSupportedKinds
go test ./internal/controller -v -run TestIsKindSupported

# Run all controller tests
go test ./internal/controller -v
```

These tests verify that common resource kinds are correctly mapped to their GVRs and that the utility functions work as expected. 