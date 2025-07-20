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

package watchdog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/madmmas/gokubedog/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var SlackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")

// PolicyViolationReportReconciler reconciles a PolicyViolationReport object
type PolicyViolationReportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyviolationreports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyviolationreports/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=watchdog.bizaikube.io,resources=policyviolationreports/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PolicyViolationReport object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *PolicyViolationReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	log := l.WithValues("violation", req.NamespacedName)
	log.Info("PolicyViolationReport controller triggered", "request", req.NamespacedName)

	var report v1alpha1.PolicyViolationReport
	if err := r.Get(ctx, req.NamespacedName, &report); err != nil {
		log.Error(err, "unable to fetch report")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Found PolicyViolationReport", "name", report.Name, "annotations", report.Annotations)

	// Avoid duplicate notifications
	if report.Annotations != nil && report.Annotations["notified"] == "true" {
		log.Info("Report already notified, skipping")
		return ctrl.Result{}, nil
	}

	log.Info("Processing new report", "webhook", SlackWebhookURL)

	if SlackWebhookURL == "" {
		log.Info("SLACK_WEBHOOK_URL not set, skipping notification")
		return ctrl.Result{}, nil
	}

	msg := formatSlackMessage(&report)

	if err := postToSlack(SlackWebhookURL, msg); err != nil {
		log.Error(err, "failed to send alert")
		return ctrl.Result{}, err
	}

	log.Info("Slack alert sent successfully")

	// Mark as notified
	if report.Annotations == nil {
		report.Annotations = map[string]string{}
	}
	report.Annotations["notified"] = "true"
	if err := r.Update(ctx, &report); err != nil {
		log.Error(err, "failed to update report with notified annotation")
		return ctrl.Result{}, err
	}
	log.Info("Updated report with notified annotation")

	return ctrl.Result{}, nil
}

func formatSlackMessage(r *v1alpha1.PolicyViolationReport) string {
	driftDetails, _ := json.MarshalIndent(r.Spec.Drift, "", "  ")

	return fmt.Sprintf(
		"*🚨 MADMMAS: Policy Violation Detected*\n*Resource:* %s/%s (%s)\n*Policy:* %s\n*Drift:*\n```%s```",
		r.Spec.ViolatedResource.Namespace,
		r.Spec.ViolatedResource.Name,
		r.Spec.ViolatedResource.Kind,
		r.Spec.ProfileName,
		string(driftDetails),
	)
}

func postToSlack(webhook, msg string) error {
	payload := map[string]string{"text": msg}
	jsonBody, _ := json.Marshal(payload)

	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook error: %s", resp.Status)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolicyViolationReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PolicyViolationReport{}).
		Named("watchdog-policyviolationreport").
		Complete(r)
}
