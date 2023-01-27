package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"testing"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"github.com/quero-edu/loki-rule-operator/internal/logger"
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
			configMapName := "loki-rule-cfg"

			BeforeEach(func() {
				lokiRule := &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Name: configMapName,
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
						Name:      configMapName,
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
				expectedVolumeName := fmt.Sprintf("%s-volume", configMapName)
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

					if resultStatefulSet.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name != configMapName {
						GinkgoWriter.Println("ConfigMap name is not test-lokirule-config")
						return false
					}

					// generated from lokirule.data
					const expectedAnnotationHash = "3e80b3778b3b03766e7be993131c0af2ad05630c5d96fb7fa132d05b77336e04"
					expectedAnnotationName := fmt.Sprintf("checksum/config-%s", configMapName)

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

			It("Should be able to handle 2 LokiRules", func() {
				lokiRule := &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule-2",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Name: "loki-rule-cfg-2",
						Data: map[string]string{
							"test2": "test2",
						},
					},
				}
				err := k8sClient.Create(context.TODO(), lokiRule)
				Expect(err).To(BeNil())

				configMap := &corev1.ConfigMap{}
				Eventually(func() bool {

					err = k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      lokiRuleConfigMapName,
						Namespace: lokiSTSNamespaceName,
					}, configMap)

					if err != nil {
						GinkgoWriter.Println("Error getting configMap: %v", err)
						return false
					}
					GinkgoWriter.Println("ConfigMap data: ", configMap.Data)
					if configMap.Data["test"] != "test" {
						GinkgoWriter.Println("ConfigMap data is not correct")
						return false
					}
					if configMap.Data["test2"] != "test2" {
						GinkgoWriter.Println("ConfigMap data is not correct")
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())
			})

		})

		Context("When a LokiRule is deleted", func() {
			var lokiRule2 = &querocomv1alpha1.LokiRule{}
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
				lokiRule2 = &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test2-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Name: "test2-lokirule-config-delete",
						Data: map[string]string{
							"test2": "test2",
						},
					},
				}
				err := k8sClient.Create(context.TODO(), lokiRule)
				Expect(err).To(BeNil())

				err = k8sClient.Create(context.TODO(), lokiRule2)
				Expect(err).To(BeNil())

				err = k8sClient.Delete(context.TODO(), lokiRule)
				Expect(err).To(BeNil())
			})

			It("Should remove the data from the configMap", func() {
				configMap := &corev1.ConfigMap{}

				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      lokiRuleConfigMapName,
						Namespace: lokiSTSNamespaceName,
					}, configMap)
					if err != nil {
						GinkgoWriter.Println("Error getting configMap, %v", err)
						return false
					}
					if reflect.DeepEqual(configMap.Data, lokiRule2.Spec.Data) {
						return true
					}
					return false
				}, timeout, interval).Should(BeTrue())
			})
		})

		Context("When a LokiRule is updated", func() {
			var newData = map[string]string{"test": "testNewValue"}
			BeforeEach(func() {
				lokiRule := &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Name: "test-lokirule-config-update",
						Data: map[string]string{
							"test": "test",
						},
					},
				}
				err := k8sClient.Create(context.TODO(), lokiRule)
				Expect(err).To(BeNil())

				lokiRule.Spec.Data = newData
				err = k8sClient.Update(context.TODO(), lokiRule)
				Expect(err).To(BeNil())
			})
			It("Should update the data in the configMap", func() {
				configMap := &corev1.ConfigMap{}
				Eventually(func() bool {
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      lokiRuleConfigMapName,
						Namespace: lokiSTSNamespaceName,
					}, configMap)
					if err != nil {
						GinkgoWriter.Println("Error getting configMap, %v", err)
						return false
					}
					if configMap.Data["test"] == newData["test"] {
						return true
					}
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

	return k8sClient.Create(context.TODO(), ns)
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
