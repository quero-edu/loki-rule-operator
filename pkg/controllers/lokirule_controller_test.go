package controllers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	"k8s.io/apimachinery/pkg/util/yaml"
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

const lokiRuleConfigMapName = "loki-rule-cfg"

var k8sClient client.Client
var testEnv *envtest.Environment

var httpServer *httptest.Server

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

	httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	lokiRuleReconcilerInstance = &LokiRuleReconciler{
		Client:                k8sClient,
		Scheme:                testEnv.Scheme,
		Logger:                logger.NewNopLogger(),
		LokiRulesPath:         lokiRuleMountPath,
		LokiLabelSelector:     selector,
		LokiNamespace:         lokiSTSNamespaceName,
		LokiRuleConfigMapName: lokiRuleConfigMapName,
		LokiURL:               httpServer.URL,
		LokiClient:            &http.Client{},
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
			var lokiRule *querocomv1alpha1.LokiRule
			configMapName := "loki-rule-cfg"

			BeforeEach(func() {
				lokiRule = &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Groups: []querocomv1alpha1.RuleGroup{
							{
								Name: "test_group",
								Rules: []querocomv1alpha1.Rule{
									{
										Record: "test_record",
										Expr:   "test_expr",
									},
								},
							},
						},
					},
				}
				err := k8sClient.Create(context.TODO(), lokiRule)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				lokiRule = &querocomv1alpha1.LokiRule{}
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
						GinkgoWriter.Printf("Error getting configMap: %v\n", err)
						return false
					}

					expectedCfgMapData := map[string]interface{}{
						"groups": []interface{}{
							map[string]interface{}{
								"name": "test_group",
								"rules": []interface{}{
									map[string]interface{}{
										"record": "test_record",
										"expr":   "test_expr",
									},
								},
							},
						},
					}

					unmarshaledCfgMapData := map[string]interface{}{}
					err = yaml.Unmarshal([]byte(configMap.Data["default-test-lokirule.yaml"]), &unmarshaledCfgMapData)
					if err != nil {
						GinkgoWriter.Printf("Error unmarshaling configMap data, %v", err)
						return false
					}

					if !reflect.DeepEqual(expectedCfgMapData, unmarshaledCfgMapData) {
						GinkgoWriter.Printf(
							"ConfigMap data does not match, Got: %v\nExpected: %v\n",
							unmarshaledCfgMapData,
							expectedCfgMapData,
						)

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
						GinkgoWriter.Printf("Error getting statefulset: %v\n", err)
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
					const expectedAnnotationHash = "3866fad9d3a968d9a28648397af7c83d57b93e629652b1962631eae56790f75a"
					expectedAnnotationName := fmt.Sprintf("checksum/config-%s", configMapName)

					if resultStatefulSet.Spec.Template.Annotations == nil {
						GinkgoWriter.Println("Annotations is not set")
						return false
					} else if resultStatefulSet.Spec.Template.Annotations[expectedAnnotationName] != expectedAnnotationHash {
						GinkgoWriter.Printf(
							"Annotation is incorrect\n\texpected: %v\n\tgot annotations: %v\n",
							map[string]string{expectedAnnotationName: expectedAnnotationHash},
							resultStatefulSet.Spec.Template.Annotations,
						)
						return false
					}

					return true
				}, timeout, interval).Should(BeTrue())
			})

			Context("When a second loki rule is created", func() {
				lokiRuleTwo := &querocomv1alpha1.LokiRule{}
				BeforeEach(func() {
					lokiRuleTwo := &querocomv1alpha1.LokiRule{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-lokirule-2",
							Namespace: namespaceName,
						},
						Spec: querocomv1alpha1.LokiRuleSpec{
							Groups: []querocomv1alpha1.RuleGroup{
								{
									Name: "default-test-lokirule-2",
									Rules: []querocomv1alpha1.Rule{
										{
											Record: "test_record2",
											Expr:   "test_expr2",
										},
									},
								},
							},
						},
					}
					err := k8sClient.Create(context.TODO(), lokiRuleTwo)
					Expect(err).To(BeNil())
				})

				AfterEach(func() {
					lokiRuleTwo = &querocomv1alpha1.LokiRule{}
					err := k8sClient.Get(context.TODO(), client.ObjectKey{
						Name:      "test-lokirule-2",
						Namespace: namespaceName,
					}, lokiRuleTwo)
					Expect(err).To(BeNil())

					err = k8sClient.Delete(context.TODO(), lokiRuleTwo)
					Expect(err).To(BeNil())
				})

				It("Should add both to the cfg map", func() {
					configMap := &corev1.ConfigMap{}
					Eventually(func() bool {
						err := k8sClient.Get(context.TODO(), client.ObjectKey{
							Name:      lokiRuleConfigMapName,
							Namespace: lokiSTSNamespaceName,
						}, configMap)

						if err != nil {
							GinkgoWriter.Printf("Error getting configMap: %v\n", err)
							return false
						}

						if len(configMap.Data) != 2 {
							GinkgoWriter.Printf("ConfigMap data length is not 2, got %v\n", len(configMap.Data))
						}

						lokiRuleOneExpectedCfgMapData := map[string]interface{}{
							"groups": []interface{}{
								map[string]interface{}{
									"name": "test_group",
									"rules": []interface{}{
										map[string]interface{}{
											"record": "test_record",
											"expr":   "test_expr",
										},
									},
								},
							},
						}

						lokiRuleOneUnmarshaledCfgMapData := map[string]interface{}{}
						err = yaml.Unmarshal(
							[]byte(configMap.Data["default-test-lokirule.yaml"]),
							&lokiRuleOneUnmarshaledCfgMapData,
						)
						if err != nil {
							GinkgoWriter.Printf("Error unmarshaling configMap data, %v", err)
							return false
						}

						if !reflect.DeepEqual(lokiRuleOneExpectedCfgMapData, lokiRuleOneUnmarshaledCfgMapData) {
							GinkgoWriter.Printf(
								"ConfigMap data from rule 1 does not match, Got: %v\nExpected: %v\n",
								lokiRuleOneUnmarshaledCfgMapData,
								lokiRuleOneExpectedCfgMapData,
							)

							return false
						}

						lokiRuleTwoExpectedCfgMapData := map[string]interface{}{
							"groups": []interface{}{
								map[string]interface{}{
									"name": "default-test-lokirule-2",
									"rules": []interface{}{
										map[string]interface{}{
											"record": "test_record2",
											"expr":   "test_expr2",
										},
									},
								},
							},
						}

						lokiRuleTwoUnmarshaledCfgMapData := map[string]interface{}{}

						err = yaml.Unmarshal(
							[]byte(configMap.Data["default-test-lokirule-2.yaml"]),
							&lokiRuleTwoUnmarshaledCfgMapData,
						)
						if err != nil {
							GinkgoWriter.Printf("Error unmarshaling configMap data, %v", err)
							return false
						}

						if !reflect.DeepEqual(lokiRuleTwoExpectedCfgMapData, lokiRuleTwoUnmarshaledCfgMapData) {
							GinkgoWriter.Printf(
								"ConfigMap data from rule 2 does not match, Got: %v\nExpected: %v\n",
								lokiRuleTwoUnmarshaledCfgMapData,
								lokiRuleTwoExpectedCfgMapData,
							)

							return false
						}

						return true
					}, timeout, interval).Should(BeTrue())
				})
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
						Groups: []querocomv1alpha1.RuleGroup{
							{
								Name: "test_group",
								Rules: []querocomv1alpha1.Rule{
									{
										Record: "test_record",
										Expr:   "test_expr",
									},
								},
							},
						},
					},
				}
				lokiRule2 = &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test2-lokirule",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Groups: []querocomv1alpha1.RuleGroup{
							{
								Name: "test_group2",
								Rules: []querocomv1alpha1.Rule{
									{
										Record: "test_record2",
										Expr:   "test_expr2",
									},
								},
							},
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

			AfterEach(func() {
				lokiRule2 = &querocomv1alpha1.LokiRule{}
				err := k8sClient.Get(context.TODO(), client.ObjectKey{
					Name:      "test2-lokirule",
					Namespace: namespaceName,
				}, lokiRule2)
				Expect(err).To(BeNil())

				err = k8sClient.Delete(context.TODO(), lokiRule2)
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
						GinkgoWriter.Printf("Error getting configMap, %v\n", err)
						return false
					}

					expectedCfgMapData := map[string]interface{}{
						"groups": []interface{}{
							map[string]interface{}{
								"name": "test_group2",
								"rules": []interface{}{
									map[string]interface{}{
										"record": "test_record2",
										"expr":   "test_expr2",
									},
								},
							},
						},
					}

					unmarshaledCfgMapData := map[string]interface{}{}
					err = yaml.Unmarshal([]byte(configMap.Data["default-test2-lokirule.yaml"]), &unmarshaledCfgMapData)
					if err != nil {
						GinkgoWriter.Printf("Error unmarshaling configMap data, %v", err)
						return false
					}

					if !reflect.DeepEqual(expectedCfgMapData, unmarshaledCfgMapData) {
						GinkgoWriter.Printf(
							"ConfigMap data does not match, Got: %v\nExpected: %v\n",
							unmarshaledCfgMapData,
							expectedCfgMapData,
						)
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())
			})
		})

		Context("When a LokiRule is updated", func() {
			BeforeEach(func() {
				lokiRule := &querocomv1alpha1.LokiRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-lokirule-update",
						Namespace: namespaceName,
					},
					Spec: querocomv1alpha1.LokiRuleSpec{
						Groups: []querocomv1alpha1.RuleGroup{
							{
								Name: "test_group",
								Rules: []querocomv1alpha1.Rule{
									{
										Record: "test_record_update",
										Expr:   "test_expr_update",
									},
								},
							},
						},
					},
				}
				err := k8sClient.Create(context.TODO(), lokiRule)
				Expect(err).To(BeNil())

				lokiRule.Spec.Groups[0].Rules[0].Expr = "test_expr_update2"

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
						GinkgoWriter.Printf("Error getting configMap, %v\n", err)
						return false
					}

					unmarshaledRuleFile := map[string][]map[string]interface{}{}
					ruleFileByteData := []byte(configMap.Data["default-test-lokirule-update.yaml"])
					err = yaml.Unmarshal(ruleFileByteData, &unmarshaledRuleFile)
					if err != nil {
						GinkgoWriter.Printf("Error unmarshaling rules, %v\n%v", err, string(ruleFileByteData))
						return false
					}

					if len(unmarshaledRuleFile) != 1 {
						GinkgoWriter.Printf(
							"RuleFile length does not match, Got: %v - Expected: %v\n",
							len(unmarshaledRuleFile),
							1,
						)
						return false
					}

					unmarshaledRules := unmarshaledRuleFile["groups"][0]["rules"].([]interface{})

					if len(unmarshaledRules) != 1 {
						GinkgoWriter.Printf("Rule length does not match, Got: %v - Expected: %v\n", len(unmarshaledRules), 1)
						return false
					}

					ruleExpression := unmarshaledRules[0].(map[string]interface{})["expr"].(string)
					if ruleExpression != "test_expr_update2" {
						GinkgoWriter.Printf(
							"Rule expr does not match (rule: %v), Got: %s\nExpected: %s\n",
							unmarshaledRules,
							ruleExpression,
							"test_expr_update2",
						)
						return false
					}

					return true
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
