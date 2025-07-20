package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	watchdogv1alpha1 "github.com/madmmas/gokubedog/api/v1alpha1"
	"github.com/madmmas/gokubedog/internal/controller"
	"github.com/madmmas/gokubedog/internal/controller/watchdog"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var (
	testEnv   *envtest.Environment
	k8sClient client.Client
	mgr       manager.Manager
	stopMgr   context.CancelFunc
	ctx       context.Context
	cfg       *rest.Config
	err       error
)

var _ = BeforeSuite(func() {
	// Set the envtest binary path
	os.Setenv("KUBEBUILDER_ASSETS", "/Users/moinuddinmasud/go/src/gokubedog/bin/k8s/1.33.0-darwin-arm64")

	Expect(os.Setenv("KUBEBUILDER_ASSETS", "../bin/k8s/1.25.0-darwin-amd64")).To(Succeed())

	ctx, stopMgr = context.WithCancel(context.Background())
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{"../../config/crd/bases"},
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	// Register scheme
	err = watchdogv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	mgr, err = manager.New(cfg, manager.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	// Register controllers with proper initialization
	policyProfileReconciler := &controller.PolicyProfileReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		DynClnt: dynamic.NewForConfigOrDie(cfg),
	}
	Expect(policyProfileReconciler.SetupWithManager(mgr)).To(Succeed())

	policyViolationReconciler := &watchdog.PolicyViolationReportReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	Expect(policyViolationReconciler.SetupWithManager(mgr)).To(Succeed())

	// Mock Slack HTTP
	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       http.NoBody,
			Header:     make(http.Header),
		}, nil
	})

	// Start the manager
	go func() {
		err := mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// Wait a moment for the manager to start
	time.Sleep(2 * time.Second)
})

var _ = AfterSuite(func() {
	stopMgr()
	_ = testEnv.Stop()
})

var _ = Describe("Controller Integration", func() {
	It("should create a PolicyViolationReport and set notified annotation on drift", func() {
		// Create PolicyProfile
		profile := &watchdogv1alpha1.PolicyProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "integration-profile",
				Namespace: "default",
			},
			Spec: watchdogv1alpha1.PolicyProfileSpec{
				Match: watchdogv1alpha1.MatchSpec{
					Kind:      "NetworkPolicy",
					Namespace: "default",
				},
				Policy: map[string]string{"foo": "bar"},
			},
		}
		Expect(k8sClient.Create(context.Background(), profile)).To(Succeed())

		// Create NetworkPolicy with drift
		np := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "networking.k8s.io/v1",
				"kind":       "NetworkPolicy",
				"metadata": map[string]interface{}{
					"name":      "np-drift",
					"namespace": "default",
					"labels":    map[string]interface{}{"foo": "not-bar"},
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
		// Use dynamic client from manager config
		dynClnt := dynamic.NewForConfigOrDie(cfg)
		_, err = dynClnt.Resource(gvr).Namespace("default").Create(context.Background(), np, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Manually trigger the PolicyProfile controller to process the drift
		fmt.Println("Manually triggering PolicyProfile controller...")
		policyProfileReconciler := &controller.PolicyProfileReconciler{
			Client:  k8sClient,
			Scheme:  scheme.Scheme,
			DynClnt: dynClnt,
		}
		_, err = policyProfileReconciler.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: client.ObjectKeyFromObject(profile),
		})
		Expect(err).NotTo(HaveOccurred())

		// Wait for PolicyViolationReport to be created
		Eventually(func() int {
			var reports watchdogv1alpha1.PolicyViolationReportList
			_ = k8sClient.List(context.Background(), &reports, client.InNamespace("default"))
			return len(reports.Items)
		}, 5*time.Second, 1*time.Second).Should(BeNumerically(">", 0))

		// Manually trigger the PolicyViolationReport controller for each report
		fmt.Println("Manually triggering PolicyViolationReport controller...")
		var reports watchdogv1alpha1.PolicyViolationReportList
		_ = k8sClient.List(context.Background(), &reports, client.InNamespace("default"))

		policyViolationReconciler := &watchdog.PolicyViolationReportReconciler{
			Client: k8sClient,
			Scheme: scheme.Scheme,
		}

		// Set webhook URL for the test
		oldWebhook := watchdog.SlackWebhookURL
		watchdog.SlackWebhookURL = "http://dummy-webhook"
		defer func() { watchdog.SlackWebhookURL = oldWebhook }()

		for _, report := range reports.Items {
			fmt.Printf("Processing report: %s\n", report.Name)
			_, err = policyViolationReconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(&report),
			})
			Expect(err).NotTo(HaveOccurred())
		}

		// Wait for PolicyViolationReport to be created and notified
		Eventually(func() bool {
			var reports watchdogv1alpha1.PolicyViolationReportList
			_ = k8sClient.List(context.Background(), &reports, client.InNamespace("default"))
			fmt.Printf("Found %d reports\n", len(reports.Items))
			if len(reports.Items) == 0 {
				return false
			}
			fmt.Printf("Report annotations: %v\n", reports.Items[0].Annotations)
			return reports.Items[0].Annotations["notified"] == "true"
		}, 10*time.Second, 1*time.Second).Should(BeTrue())
	})
})
