package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"testing"

	"github.com/go-kit/log"
	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestLokiRuleController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LokiRuleController Suite")
}

const timeout = time.Second * 1
const interval = time.Millisecond * 250

const lokiRuleMountPath = "/etc/loki/rules"
const namespaceName = "default"

var k8sClient client.Client
var testEnv *envtest.Environment

var lokiStatefulSet *appsv1.StatefulSet
var lokiRuleReconcilerInstance *LokiRuleReconciler

var _ = BeforeSuite(func() {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())

	err = querocomv1alpha1.AddToScheme(testEnv.Scheme)
	Expect(err).ToNot(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: testEnv.Scheme})

	Expect(err).ToNot(HaveOccurred())

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	k8sClient.Create(context.TODO(), namespace)

	labels := map[string]string{
		"app": "loki",
	}

	lokiStatefulSet, err = createStatefulSet(k8sClient, namespaceName, labels)
	Expect(err).ToNot(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: testEnv.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	lokiRuleReconcilerInstance = &LokiRuleReconciler{
		Client:                  k8sClient,
		Scheme:                  testEnv.Scheme,
		Logger:                  log.NewNopLogger(),
		LokiStatefulSetInstance: lokiStatefulSet,
		LokiRulesPath:           lokiRuleMountPath,
	}

	err = (lokiRuleReconcilerInstance).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())

		gexec.KillAndWait(4 * time.Second)
		err := testEnv.Stop()
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = Describe("LokiRuleController", func() {
	Describe("Reconcile Create", func() {
		Context("When a LokiRule is created", func() {
			BeforeEach(func() {
				lokiRule := &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Name: "test-lokirule-config",
						Data: map[string]string{
							"test": "test",
						},
					},
				}
				err := k8sClient.Create(context.TODO(), lokiRule)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				lokiRule := &querocomv1alpha1.LokiRule{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{
					Name:      "test-lokirule",
					Namespace: namespaceName,
				}, lokiRule)

				Expect(err).To(BeNil())

				err = k8sClient.Delete(context.TODO(), lokiRule)
				Expect(err).To(BeNil())
			})

			It("Should create the configMap", func() {
				configMap := &corev1.ConfigMap{}

				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      "test-lokirule-config",
						Namespace: namespaceName,
					}, configMap)
					if err != nil {
						GinkgoWriter.Println("Error getting configMap: %v", err)
						return false
					}

					if configMap.Data["test"] != "test" {
						GinkgoWriter.Println("ConfigMap data is not correct")
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())
			})

			It("Should mount the configMap", func() {
				expectedVolumeName := fmt.Sprintf("%s-volume", "test-lokirule-config")
				resultStatefulSet := &appsv1.StatefulSet{}

				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      lokiStatefulSet.Name,
						Namespace: namespaceName,
					}, resultStatefulSet)

					if err != nil {
						return false
					}

					if len(resultStatefulSet.Spec.Template.Spec.Volumes) != 1 {
						GinkgoWriter.Println("Volumes length is not 1")
						return false
					}

					if len(resultStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts) != 1 {
						GinkgoWriter.Println("VolumeMounts length is not 1")
						return false
					}

					if resultStatefulSet.Spec.Template.Spec.Volumes[0].Name != expectedVolumeName {
						GinkgoWriter.Println("Volume name is not %s", expectedVolumeName)
						return false
					}

					if resultStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name != expectedVolumeName {
						GinkgoWriter.Println("VolumeMount name is not %s", expectedVolumeName)
						return false
					}

					if resultStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath != lokiRuleMountPath {
						GinkgoWriter.Println("VolumeMount path is not %s", lokiRuleMountPath)
						return false
					}

					if resultStatefulSet.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name != "test-lokirule-config" {
						GinkgoWriter.Println("ConfigMap name is not test-lokirule-config")
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())
			})
		})
	})
})

func createStatefulSet(k8sClient client.Client, namespace string, labels map[string]string) (*appsv1.StatefulSet, error) {
	lokiStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "loki", Image: "grafana/loki:2.2.1"}}}},
		},
	}

	err := k8sClient.Create(context.TODO(), lokiStatefulSet)
	if err != nil {
		return nil, err
	}

	return lokiStatefulSet, nil
}
