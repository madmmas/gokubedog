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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	watchdogv1alpha1 "github.com/madmmas/gokubedog/api/v1alpha1"
	"k8s.io/client-go/dynamic"
)

var _ = Describe("PolicyProfile Controller", func() {
	const (
		resourceName = "test-resource"
		ns           = "default"
	)
	ctx := context.Background()

	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: ns,
	}

	// Helper to create PolicyProfile
	createPolicyProfile := func(policy map[string]string, matchKind, matchNS string) *watchdogv1alpha1.PolicyProfile {
		profile := &watchdogv1alpha1.PolicyProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: ns,
			},
			Spec: watchdogv1alpha1.PolicyProfileSpec{
				Match: watchdogv1alpha1.MatchSpec{
					Kind:      matchKind,
					Namespace: matchNS,
				},
				Policy: policy,
			},
		}
		Expect(k8sClient.Create(ctx, profile)).To(Succeed())
		return profile
	}

	// Helper to create NetworkPolicy
	createNetworkPolicy := func(name string, labels map[string]string) {
		np := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "networking.k8s.io/v1",
				"kind":       "NetworkPolicy",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": ns,
					"labels":    labels,
				},
				"spec": map[string]interface{}{
					"podSelector": map[string]interface{}{},
					"policyTypes": []interface{}{"Ingress"},
				},
			},
		}
		gvr := schema.GroupVersionResource{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "networkpolicies",
		}
		_, err := dynamic.NewForConfigOrDie(cfg).Resource(gvr).Namespace(ns).Create(ctx, np, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	}

	// Helper to get violation reports
	getReports := func() []watchdogv1alpha1.PolicyViolationReport {
		var reports watchdogv1alpha1.PolicyViolationReportList
		_ = k8sClient.List(ctx, &reports, client.InNamespace(ns))
		return reports.Items
	}

	AfterEach(func() {
		// Cleanup PolicyProfile
		profile := &watchdogv1alpha1.PolicyProfile{}
		_ = k8sClient.Get(ctx, typeNamespacedName, profile)
		_ = k8sClient.Delete(ctx, profile)
		// Cleanup PolicyViolationReports
		var reports watchdogv1alpha1.PolicyViolationReportList
		_ = k8sClient.List(ctx, &reports, client.InNamespace(ns))
		for _, r := range reports.Items {
			_ = k8sClient.Delete(ctx, &r)
		}
		// Cleanup NetworkPolicies
		gvr := schema.GroupVersionResource{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "networkpolicies",
		}
		list, _ := dynamic.NewForConfigOrDie(cfg).Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{})
		for _, item := range list.Items {
			_ = dynamic.NewForConfigOrDie(cfg).Resource(gvr).Namespace(ns).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		}
	})

	It("should create a PolicyViolationReport when drift is detected", func() {
		controllerReconciler := &PolicyProfileReconciler{
			Client:  k8sClient,
			Scheme:  k8sClient.Scheme(),
			DynClnt: dynamic.NewForConfigOrDie(cfg),
		}
		createPolicyProfile(map[string]string{"foo": "bar"}, "NetworkPolicy", ns)
		createNetworkPolicy("np-drift", map[string]string{"foo": "not-bar"})
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
		Expect(err).NotTo(HaveOccurred())
		By("Expecting a violation report to be created")
		Eventually(func() int { return len(getReports()) }, 5*time.Second).Should(Equal(1))
		// Optionally check report content
		reports := getReports()
		Expect(reports[0].Spec.ProfileName).To(Equal(resourceName))
		Expect(reports[0].Spec.Drift).To(HaveKey("foo"))
	})

	It("should not create a PolicyViolationReport when there is no drift", func() {
		controllerReconciler := &PolicyProfileReconciler{
			Client:  k8sClient,
			Scheme:  k8sClient.Scheme(),
			DynClnt: dynamic.NewForConfigOrDie(cfg),
		}
		createPolicyProfile(map[string]string{"foo": "bar"}, "NetworkPolicy", ns)
		createNetworkPolicy("np-match", map[string]string{"foo": "bar"})
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
		Expect(err).NotTo(HaveOccurred())
		Consistently(func() int { return len(getReports()) }, 2*time.Second).Should(Equal(0))
	})

	It("should not create a PolicyViolationReport if no matching resources exist", func() {
		controllerReconciler := &PolicyProfileReconciler{
			Client:  k8sClient,
			Scheme:  k8sClient.Scheme(),
			DynClnt: dynamic.NewForConfigOrDie(cfg),
		}
		createPolicyProfile(map[string]string{"foo": "bar"}, "NetworkPolicy", ns)
		// No NetworkPolicy created
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
		Expect(err).NotTo(HaveOccurred())
		Consistently(func() int { return len(getReports()) }, 2*time.Second).Should(Equal(0))
	})
})
