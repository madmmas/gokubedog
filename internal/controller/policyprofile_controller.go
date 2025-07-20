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
	"context"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	watchdogv1alpha1 "github.com/madmmas/gokubedog/api/v1alpha1"
)

// PolicyProfileReconciler reconciles a PolicyProfile object
type PolicyProfileReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	DynClnt dynamic.Interface
	// Cache for discovered API resources
	apiResources map[string]schema.GroupVersionResource
	// Discovery client for dynamic API discovery
	discoveryClient discovery.DiscoveryInterface
}

// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyprofiles/finalizers,verbs=update

// discoverGVRFromKind attempts to discover the GVR for a kind dynamically
func (r *PolicyProfileReconciler) discoverGVRFromKind(ctx context.Context, kind string) (schema.GroupVersionResource, error) {
	// First try the static mapping
	if gvr, err := GetGVRFromKind(kind); err == nil {
		return gvr, nil
	}

	// If not found in static mapping, try to discover dynamically
	if r.apiResources == nil {
		r.apiResources = make(map[string]schema.GroupVersionResource)
	}

	// Check if we already discovered this kind
	if gvr, exists := r.apiResources[kind]; exists {
		return gvr, nil
	}

	// Get the discovery client to discover API resources
	if r.discoveryClient == nil {
		return schema.GroupVersionResource{}, fmt.Errorf("discovery client not initialized")
	}

	// Get all API groups
	apiGroups, err := r.discoveryClient.ServerGroups()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("failed to get server groups: %w", err)
	}

	// Search through all API groups for the kind
	for _, apiGroup := range apiGroups.Groups {
		for _, version := range apiGroup.Versions {
			// Get API resources for this group version
			apiResources, err := r.discoveryClient.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				// Skip if we can't get resources for this group version
				continue
			}

			// Look for the kind in this group version
			for _, resource := range apiResources.APIResources {
				if resource.Kind == kind {
					gvr := schema.GroupVersionResource{
						Group:    apiGroup.Name,
						Version:  version.Version,
						Resource: resource.Name,
					}
					// Cache the result
					r.apiResources[kind] = gvr
					return gvr, nil
				}
			}
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("kind %s not found in any API group", kind)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PolicyProfile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *PolicyProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	var profile watchdogv1alpha1.PolicyProfile
	if err := r.Get(ctx, req.NamespacedName, &profile); err != nil {
		l.Error(err, "unable to fetch PolicyProfile")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Step 1: Derive GroupVersionResource for resource kind dynamically
	kind := profile.Spec.Match.Kind
	nsPattern := profile.Spec.Match.Namespace
	desiredPolicy := profile.Spec.Policy

	gvr, err := r.discoverGVRFromKind(ctx, kind)
	if err != nil {
		l.Error(err, "unsupported resource kind", "kind", kind)
		return ctrl.Result{}, err
	}

	l.Info("Processing PolicyProfile", "kind", kind, "gvr", gvr, "namespacePattern", nsPattern)

	// Step 2: List all matching resources
	resList, err := r.DynClnt.Resource(gvr).Namespace("").List(ctx, v1.ListOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed listing target resources: %w", err)
	}

	for _, item := range resList.Items {
		if !matchNamespace(item.GetNamespace(), nsPattern) {
			continue
		}

		// Step 3: Drift detection
		labels := item.GetLabels()
		if drift := detectDrift(labels, desiredPolicy); len(drift) > 0 {
			l.Info("Policy drift detected", "resource", item.GetName(), "namespace", item.GetNamespace(), "drift", drift)
			// TODO: emit PolicyViolationReport or alert
		}
	}

	// Step 4: Update status
	profile.Status.LastChecked = v1.Now()
	if err := r.Status().Update(ctx, &profile); err != nil {
		l.Error(err, "unable to update PolicyProfile status")
	}

	return ctrl.Result{RequeueAfter: 1 * time.Hour}, nil
}

// Simple string glob match (basic wildcard support)
func matchNamespace(actual, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(actual, strings.TrimSuffix(pattern, "*"))
	}
	return actual == pattern
}

// Compare desired policy with live labels (can expand later)
func detectDrift(actualLabels map[string]string, desired map[string]string) map[string]string {
	drift := map[string]string{}
	for k, v := range desired {
		actualVal, exists := actualLabels[k]
		if !exists || actualVal != v {
			drift[k] = fmt.Sprintf("Expected: %s, Got: %s", v, actualVal)
		}
	}
	return drift
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolicyProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.DynClnt = dynamic.NewForConfigOrDie(mgr.GetConfig())
	r.discoveryClient = discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig())

	return ctrl.NewControllerManagedBy(mgr).
		For(&watchdogv1alpha1.PolicyProfile{}).
		Named("policyprofile").
		Complete(r)
}
