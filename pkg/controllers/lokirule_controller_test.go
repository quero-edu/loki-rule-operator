package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"testing"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"github.com/quero-edu/loki-rule-operator/internal/logger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
const lokiSTSNamespaceName = "loki"

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

	labels := map[string]string{
		"app": "loki",
	}

	err = createNamespace(k8sClient, lokiSTSNamespaceName)
	Expect(err).ToNot(HaveOccurred())

	lokiStatefulSet, err = createStatefulSet(k8sClient, lokiSTSNamespaceName, labels)
	Expect(err).ToNot(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: testEnv.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	lokiRuleReconcilerInstance = &LokiRuleReconciler{
		Client:            k8sClient,
		Scheme:            testEnv.Scheme,
		Logger:            logger.NewNopLogger(),
		LokiRulesPath:     lokiRuleMountPath,
		LokiLabelSelector: selector,
		LokiNamespace:     lokiSTSNamespaceName,
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
	Describe("Reconcile", func() {
		Context("When a LokiRule is created", func() {
			const cfgMapName = "test-lokirule"
			const expectedConfigMapName = "test-lokirule-default"

			BeforeEach(func() {
				lokiRule := &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Name: cfgMapName,
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
						Name:      expectedConfigMapName,
						Namespace: lokiSTSNamespaceName,
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

			It("Should mount the configMap and annotate the statefulset", func() {
				expectedVolumeName := fmt.Sprintf("%s-volume", expectedConfigMapName)
				resultStatefulSet := &appsv1.StatefulSet{}

				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      lokiStatefulSet.Name,
						Namespace: lokiSTSNamespaceName,
					}, resultStatefulSet)

					if err != nil {
						GinkgoWriter.Println("Error getting statefulset: %v", err)
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

					if resultStatefulSet.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name != expectedConfigMapName {
						GinkgoWriter.Println("ConfigMap name is not test-lokirule-config")
						return false
					}

					// generated from lokirule.data
					const expectedAnnotationHash = "3e80b3778b3b03766e7be993131c0af2ad05630c5d96fb7fa132d05b77336e04"
					expectedAnnotationName := fmt.Sprintf("checksum/config-%s", expectedConfigMapName)

					if resultStatefulSet.Spec.Template.Annotations == nil {
						GinkgoWriter.Println("Annotations is not set")
						return false
					} else if resultStatefulSet.Spec.Template.Annotations[expectedAnnotationName] != expectedAnnotationHash {
						GinkgoWriter.Printf(
							"\nAnnotation is incorrect\n\texpected: %v\n\tgot annotations: %v",
							map[string]string{expectedAnnotationName: expectedAnnotationHash},
							resultStatefulSet.Spec.Template.Annotations,
						)
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())
			})
		})

		Context("When a LokiRule is deleted", func() {
			BeforeEach(func() {
				lokiRule := &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Name: "test-lokirule-config-delete",
						Data: map[string]string{
							"test": "test",
						},
					},
				}
				err := k8sClient.Create(context.TODO(), lokiRule)
				Expect(err).To(BeNil())

				lokiRule = &querocomv1alpha1.LokiRule{}
				err = k8sClient.Get(context.TODO(), client.ObjectKey{
					Name:      "test-lokirule",
					Namespace: namespaceName,
				}, lokiRule)

				Expect(err).To(BeNil())

				err = k8sClient.Delete(context.TODO(), lokiRule)
				Expect(err).To(BeNil())
			})

			It("Should delete the configMap", func() {
				configMap := &corev1.ConfigMap{}

				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      "test-lokirule-config-delete",
						Namespace: lokiSTSNamespaceName,
					}, configMap)
					if err != nil {
						if errors.IsNotFound(err) {
							return true
						}
						GinkgoWriter.Println("Error getting configMap, %v", err)
						return false
					}
					GinkgoWriter.Println("ConfigMap still exists")
					return false
				}, timeout, interval).Should(BeTrue())
			})

			It("Should delete the volume and annotations", func() {
				resultStatefulSet := &appsv1.StatefulSet{}
				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      lokiStatefulSet.Name,
						Namespace: lokiSTSNamespaceName,
					}, resultStatefulSet)
					if err != nil {
						GinkgoWriter.Println("Error getting statefulSet, %v", err)
						return false
					}

					if len(resultStatefulSet.Spec.Template.Spec.Volumes) == 0 &&
						len(resultStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts) == 0 &&
						len(resultStatefulSet.Spec.Template.Annotations) == 0 {
						return true
					}

					GinkgoWriter.Println("Volume still exists")
					return false
				}, timeout, interval).Should(BeTrue())
			})

		})
	})
})

func createNamespace(k8sClient client.Client, namespace string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	err := k8sClient.Create(context.TODO(), ns)
	if err != nil {
		return err
	}

	return nil
}

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
