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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
}

// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyprofiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyprofiles/finalizers,verbs=update

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

	// Step 1: Derive GroupVersionResource for resource kind
	kind := profile.Spec.Match.Kind
	nsPattern := profile.Spec.Match.Namespace
	desiredPolicy := profile.Spec.Policy

	gvr := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "networkpolicies", // TODO: map kind → resource name dynamically
	}

	// Step 2: List all matching resources
	resList, err := r.DynClnt.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
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

			// Emit PolicyViolationReport
			report := &watchdogv1alpha1.PolicyViolationReport{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "violation-",
					Namespace:    item.GetNamespace(),
				},
				Spec: watchdogv1alpha1.PolicyViolationReportSpec{
					ViolatedResource: struct {
						Kind      string `json:"kind"`
						Name      string `json:"name"`
						Namespace string `json:"namespace"`
					}{
						Kind:      kind,
						Name:      item.GetName(),
						Namespace: item.GetNamespace(),
					},
					ProfileName: profile.Name,
					Drift:       drift,
				},
			}

			if err := r.Client.Create(ctx, report); err != nil {
				l.Error(err, "unable to create PolicyViolationReport")
			} else {
				l.Info("Created PolicyViolationReport", "name", report.Name)
			}
		}
	}

	// Step 4: Update status
	profile.Status.LastChecked = metav1.Now()
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
	// return ctrl.NewControllerManagedBy(mgr).
	//     For(&watchdogv1alpha1.PolicyProfile{}).
	//     Complete(r)

	return ctrl.NewControllerManagedBy(mgr).
		For(&watchdogv1alpha1.PolicyProfile{}).
		Named("policyprofile").
		Complete(r)
}
