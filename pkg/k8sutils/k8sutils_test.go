package k8sutils

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestK8sUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "K8sUtils Suite")
}

const NAMESPACE = "default"

var k8sClient client.Client
var testEnv *envtest.Environment

var _ = BeforeSuite(func() {
	testEnv = &envtest.Environment{}

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: testEnv.Scheme})

	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("K8sutils", func() {
	Describe("TestCreateConfigMap", func() {
		It("should create a ConfigMap with the given data and labels", func() {
			configMapName := "test-configmap-create"
			configMapLabels := map[string]string{"lfoo": "lbar"}
			_, err := CreateConfigMap(k8sClient, NAMESPACE, configMapName, configMapLabels, Options{})
			Expect(err).To(BeNil())

			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      configMapName,
				Namespace: NAMESPACE,
			}, configMap)

			Expect(err).To(BeNil())
			Expect(configMap.Name).To(Equal(configMapName))
			Expect(configMap.Namespace).To(Equal(NAMESPACE))
			Expect(configMap.Labels).To(Equal(configMapLabels))
		})
	})
	Describe("AddToConfigMap", func() {
		It("should update the ConfigMap with new data", func() {
			configMapName := "test-configmap-add-update"
			configMapData := map[string]string{"foo": "bar"}
			configMapLabels := map[string]string{"lfoo": "lbar"}

			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: NAMESPACE,
					Labels:    configMapLabels,
				},
				Data: configMapData,
			}

			err := k8sClient.Create(context.TODO(), configMap)
			Expect(err).To(BeNil())

			newConfigMapData := map[string]string{"baz": "foo"}
			_, err = AddToConfigMap(
				k8sClient,
				NAMESPACE,
				configMapName,
				newConfigMapData,
				Options{},
			)

			Expect(err).To(BeNil())

			configMap = &corev1.ConfigMap{}
			err = k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      configMapName,
				Namespace: NAMESPACE,
			}, configMap)

			expectedConfigMapData := map[string]string{"foo": "bar", "baz": "foo"}
			Expect(err).To(BeNil())
			Expect(configMap.Data).To(Equal(expectedConfigMapData))
			Expect(configMap.Namespace).To(Equal(NAMESPACE))
			Expect(configMap.Name).To(Equal(configMapName))
			Expect(configMap.Labels).To(Equal(configMapLabels))
		})
	})

	Describe("RemoveFromConfigMap", func() {
		It("should remove the data from configMap", func() {
			configMapName := "test-configmap-rm-update"
			configMapData := map[string]string{"foo": "bar", "baz": "foo"}
			configMapLabels := map[string]string{"lfoo": "lbar"}

			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: NAMESPACE,
					Labels:    configMapLabels,
				},
				Data: configMapData,
			}

			err := k8sClient.Create(context.TODO(), configMap)
			Expect(err).To(BeNil())

			removeConfigMapData := map[string]string{"foo": "bar"}
			_, err = RemoveFromConfigMap(
				k8sClient,
				NAMESPACE,
				configMapName,
				removeConfigMapData,
				Options{},
			)

			Expect(err).To(BeNil())

			configMap = &corev1.ConfigMap{}
			err = k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      configMapName,
				Namespace: NAMESPACE,
			}, configMap)

			expectedConfigMapData := map[string]string{"baz": "foo"}
			Expect(err).To(BeNil())
			Expect(configMap.Data).To(Equal(expectedConfigMapData))
			Expect(configMap.Namespace).To(Equal(NAMESPACE))
			Expect(configMap.Name).To(Equal(configMapName))
			Expect(configMap.Labels).To(Equal(configMapLabels))
		})
	})

	Describe("TestGetStatefulSet", func() {
		var statefulSet *appsv1.StatefulSet
		var err error

		BeforeEach(func() {
			statefulSet, err = createStatefulSet()
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			_, err = deleteStatefulSet()
			Expect(err).To(BeNil())
		})

		Context("With matching label selector", func() {
			It("Should return the statefulSet", func() {
				labelSelector := metav1.LabelSelector{
					MatchLabels: statefulSet.Labels,
				}

				resultStatefulSet, err := GetStatefulSet(k8sClient, &labelSelector, NAMESPACE, Options{})
				Expect(err).To(BeNil())
				Expect(resultStatefulSet).To(Equal(statefulSet))
			})
		})

		Context("With non-matching label selector", func() {
			It("Should return an error", func() {
				nonMatchingLabels := map[string]string{
					"app.kubernetes.io/name": "not-loki",
				}

				nonMatchingLabelSelector := metav1.LabelSelector{
					MatchLabels: nonMatchingLabels,
				}

				emptyResult, err := GetStatefulSet(k8sClient, &nonMatchingLabelSelector, NAMESPACE, Options{})
				Expect(err.Error()).To(Equal("no statefulSets found"))
				Expect(emptyResult).To(BeNil())
			})
		})
	})

	Describe("MountConfigMap", func() {
		const mountPath = "/etc/config"

		var statefulSet *appsv1.StatefulSet
		var err error

		configMapName := "loki-config"
		configMapData := map[string]string{"foo": "bar"}
		configMapLabels := map[string]string{"app.kubernetes.io/name": "loki"}

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: NAMESPACE,
				Labels:    configMapLabels,
			},
			Data: configMapData,
		}

		BeforeEach(func() {
			statefulSet, err = createStatefulSet()
			Expect(err).To(BeNil())

			err = k8sClient.Create(context.TODO(), configMap)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			_, err = deleteStatefulSet()
			Expect(err).To(BeNil())

			err = k8sClient.Delete(context.TODO(), configMap)
			Expect(err).To(BeNil())
		})

		It("Should mount the configMap", func() {
			err = MountConfigMap(k8sClient, NAMESPACE, configMapName, mountPath, statefulSet, Options{})
			Expect(err).To(BeNil())

			updatedStatefulSet := &appsv1.StatefulSet{}
			err = k8sClient.Get(context.TODO(), types.NamespacedName{
				Name:      statefulSet.Name,
				Namespace: statefulSet.Namespace,
			}, updatedStatefulSet)
			Expect(err).To(BeNil())

			Expect(
				updatedStatefulSet.Spec.Template.Spec.Volumes[0].Name,
			).To(Equal(fmt.Sprintf("%s-volume", configMapName)))
			Expect(
				updatedStatefulSet.Spec.Template.Spec.Volumes[0].ConfigMap.LocalObjectReference.Name,
			).To(Equal(configMapName))

			Expect(
				updatedStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name,
			).To(Equal(fmt.Sprintf("%s-volume", configMapName)))
			Expect(updatedStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath).To(Equal(mountPath))

			// Hashed data using private function in k8sutil
			const expectedAnnotationHash = "7a38bf81f383f69433ad6e900d35b3e2385593f76a7b7ab5d4355b8ba41ee24b"
			expectedAnnotationName := fmt.Sprintf("checksum/config-%s", configMapName)

			Expect(
				updatedStatefulSet.Spec.Template.Annotations,
			).To(HaveKeyWithValue(expectedAnnotationName, expectedAnnotationHash))
		})
	})
})

func createStatefulSet() (*appsv1.StatefulSet, error) {
	labels := map[string]string{
		"app.kubernetes.io/name": "loki",
	}

	lokiStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: NAMESPACE,
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

func deleteStatefulSet() (noop bool, returnErr error) {
	lokiStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: NAMESPACE,
		},
	}

	err := k8sClient.Delete(context.TODO(), lokiStatefulSet)

	if err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	return false, nil
}
