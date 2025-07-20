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
	"context"

	"net/http"

	"github.com/madmmas/gokubedog/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

var testScheme = runtime.NewScheme()
var _ = func() bool {
	_ = v1alpha1.AddToScheme(testScheme)
	return true
}()

var _ = Describe("PolicyViolationReport Controller", func() {
	Context("When reconciling a resource", func() {

		It("should successfully reconcile the resource", func() {

			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	It("should set notified annotation after reconciliation", func() {
		cl := fake.NewClientBuilder().WithScheme(testScheme).Build()
		r := &PolicyViolationReportReconciler{Client: cl}
		report := &v1alpha1.PolicyViolationReport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-report",
				Namespace: "default",
			},
			Spec: v1alpha1.PolicyViolationReportSpec{
				ProfileName: "test-profile",
				Drift:       map[string]string{"foo": "bar"},
				ViolatedResource: v1alpha1.ViolatedResourceSpec{
					Kind:      "NetworkPolicy",
					Name:      "np-drift",
					Namespace: "default",
				},
			},
		}
		_ = cl.Create(context.Background(), report)
		// Patch the webhook variable to a dummy value for the test
		oldWebhook := slackWebhookURL
		slackWebhookURL = "http://dummy-webhook"
		defer func() { slackWebhookURL = oldWebhook }()
		// Mock HTTP transport
		originalTransport := http.DefaultTransport
		http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		})
		defer func() { http.DefaultTransport = originalTransport }()
		_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(report)})
		Expect(err).NotTo(HaveOccurred())
		_ = cl.Get(context.Background(), client.ObjectKeyFromObject(report), report)
		Expect(report.Annotations).To(HaveKeyWithValue("notified", "true"))
	})

	It("should not send notification if already notified", func() {
		cl := fake.NewClientBuilder().WithScheme(testScheme).Build()
		r := &PolicyViolationReportReconciler{Client: cl}
		report := &v1alpha1.PolicyViolationReport{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-report-idem",
				Namespace:   "default",
				Annotations: map[string]string{"notified": "true"},
			},
			Spec: v1alpha1.PolicyViolationReportSpec{
				ProfileName: "test-profile",
				Drift:       map[string]string{"foo": "bar"},
				ViolatedResource: v1alpha1.ViolatedResourceSpec{
					Kind:      "NetworkPolicy",
					Name:      "np-drift",
					Namespace: "default",
				},
			},
		}
		_ = cl.Create(context.Background(), report)
		_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(report)})
		Expect(err).NotTo(HaveOccurred())
		_ = cl.Get(context.Background(), client.ObjectKeyFromObject(report), report)
		Expect(report.Annotations).To(HaveKeyWithValue("notified", "true"))
	})

	It("should not panic or send notification if webhook is missing", func() {
		cl := fake.NewClientBuilder().WithScheme(testScheme).Build()
		r := &PolicyViolationReportReconciler{Client: cl}
		report := &v1alpha1.PolicyViolationReport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-report-no-webhook",
				Namespace: "default",
			},
			Spec: v1alpha1.PolicyViolationReportSpec{
				ProfileName: "test-profile",
				Drift:       map[string]string{"foo": "bar"},
				ViolatedResource: v1alpha1.ViolatedResourceSpec{
					Kind:      "NetworkPolicy",
					Name:      "np-drift",
					Namespace: "default",
				},
			},
		}
		_ = cl.Create(context.Background(), report)
		// Temporarily patch the webhook variable if needed, or rely on empty string logic
		_, err := r.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(report)})
		Expect(err).NotTo(HaveOccurred())
		_ = cl.Get(context.Background(), client.ObjectKeyFromObject(report), report)
		Expect(report.Annotations).To(BeNil())
	})
})
