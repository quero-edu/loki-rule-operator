package k8sutils

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const NAMESPACE = "default"

var k8sClient client.Client
var testEnv *envtest.Environment

func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{}

	cfg, err := testEnv.Start()
	if err != nil {
		panic(err)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})

	if err != nil {
		panic(err)
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: NAMESPACE,
		},
	}

	k8sClient.Create(context.TODO(), namespace)

	m.Run()

	testEnv.Stop()
}

func TestCreateOrUpdateConfigMap(t *testing.T) {
	// Create a configMap
	configMapName := "test-configmap"
	configMapData := map[string]string{"foo": "bar"}
	configMapLabels := map[string]string{"lfoo": "lbar"}

	_, err := CreateOrUpdateConfigMap(k8sClient, NAMESPACE, configMapName, configMapData, configMapLabels, Options{})
	if err != nil {
		t.Errorf("CreateOrUpdateConfigMap() error = %v", err)
		return
	}

	// Asserts that the configMap was created
	configMap := &corev1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      configMapName,
		Namespace: NAMESPACE,
	}, configMap)

	if err != nil {
		t.Errorf("CreateOrUpdateConfigMap() error = %v", err)
		return
	}

	if configMap.Name != configMapName {
		t.Errorf(
			"CreateOrUpdateConfigMap() error = ConfigMap was not created. Expected name to be '%s', got '%s'",
			configMapName,
			configMap.Name,
		)
		return
	}

	if !reflect.DeepEqual(configMap.Data, configMapData) {
		t.Errorf(
			"CreateOrUpdateConfigMap() error = ConfigMap was not created. Expected configMap.Data to be %s, got '%s'",
			configMapData,
			configMap.Data,
		)
	}

	if !reflect.DeepEqual(configMap.Labels, configMapLabels) {
		t.Errorf(
			"CreateOrUpdateConfigMap() error = ConfigMap was not created. Expected configMap.Labels to be %s, got '%s'",
			configMapLabels,
			configMap.Labels,
		)
	}

	// Update a configMap
	configMapData = map[string]string{"baz": "foo"}
	_, err = CreateOrUpdateConfigMap(k8sClient, NAMESPACE, configMapName, configMapData, configMapLabels, Options{})
	if err != nil {
		t.Errorf("CreateOrUpdateConfigMap() error = %v", err)
		return
	}

	// Asserts that the configMap was updated
	configMap = &corev1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      configMapName,
		Namespace: NAMESPACE,
	}, configMap)

	if err != nil {
		t.Errorf("CreateOrUpdateConfigMap() error = %v", err)
		return
	}

	if !reflect.DeepEqual(configMap.Data, configMapData) {
		t.Errorf(
			"CreateOrUpdateConfigMap() error = ConfigMap was not updated, expected configMap.Data to be %s, got '%s'",
			configMapData,
			configMap.Data,
		)
		return
	}
}

func TestGetLokiStatefulSetInstance(t *testing.T) {
	_, err := deleteLokiStatefulSet()
	if err != nil {
		t.Errorf("GetLokiStatefulSetInstance() setup error = %v", err)
		return
	}

	lokiStatefulSet, err := createLokiStatefulset()
	if err != nil {
		t.Errorf("GetLokiStatefulSetInstance() setup error = %v", err)
		return
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: lokiStatefulSet.Labels,
	}

	lokiStatefulSetInstance, err := GetLokiStatefulSetInstance(k8sClient, &labelSelector, NAMESPACE, Options{})

	if err != nil {
		t.Errorf("GetLokiStatefulSetInstance() error = %v", err)
		return
	}

	if lokiStatefulSetInstance.Name != lokiStatefulSet.Name {
		t.Errorf(
			"GetLokiStatefulSetInstance() error = Expected LokiStatefulSet name to be '%s', got '%s'",
			lokiStatefulSet.Name,
			lokiStatefulSetInstance.Name,
		)
		return
	}

	if lokiStatefulSetInstance.Namespace != lokiStatefulSet.Namespace {
		t.Errorf(
			"GetLokiStatefulSetInstance() error = Expected LokiStatefulSet namespace to be '%s', got '%s'",
			lokiStatefulSet.Namespace,
			lokiStatefulSetInstance.Namespace,
		)
		return
	}

	if !reflect.DeepEqual(lokiStatefulSetInstance.Labels, lokiStatefulSet.Labels) {
		t.Errorf(
			"GetLokiStatefulSetInstance() error = Expected LokiStatefulSet labels to be %s, got '%s'",
			lokiStatefulSet.Labels,
			lokiStatefulSetInstance.Labels,
		)
		return
	}

	nonMatchingLabels := map[string]string{
		"app.kubernetes.io/name": "not-loki",
	}

	clearSingletons()

	nonMatchingLabelSelector := metav1.LabelSelector{
		MatchLabels: nonMatchingLabels,
	}

	_, err = GetLokiStatefulSetInstance(k8sClient, &nonMatchingLabelSelector, NAMESPACE, Options{})

	expectedError := "no statefulSets found"
	if err.Error() != expectedError {
		t.Errorf("GetLokiStatefulSetInstance() expected '%s' error, got '%v'", expectedError, err)
		return
	}
}

func TestMountConfigMap(t *testing.T) {
	configMapName := "loki-config"
	configMapData := map[string]string{"foo": "bar"}
	configMapLabels := map[string]string{"app.kubernetes.io/name": "loki"}

	configMap, err := CreateOrUpdateConfigMap(k8sClient, NAMESPACE, configMapName, configMapData, configMapLabels, Options{})
	if err != nil {
		t.Errorf("MountConfigMap() setup error = %v", err)
		return
	}

	_, err = deleteLokiStatefulSet()
	if err != nil {
		t.Errorf("GetLokiStatefulSetInstance() setup error = %v", err)
		return
	}

	lokiStatefulSet, err := createLokiStatefulset()
	if err != nil {
		t.Errorf("MountConfigMap() setup error = %v", err)
		return
	}

	mountPath := "/var/loki"

	err = MountConfigMap(k8sClient, configMap, mountPath, lokiStatefulSet, Options{})
	if err != nil {
		t.Errorf("MountConfigMap() error = %v", err)
		return
	}

	clearSingletons()
	lokiStatefulSetInstance, err := GetLokiStatefulSetInstance(k8sClient, &metav1.LabelSelector{MatchLabels: lokiStatefulSet.Labels}, NAMESPACE, Options{})
	if err != nil {
		t.Errorf("MountConfigMap() setup error = %v", err)
		return
	}

	lokiStatefulSetInstanceVolumeMounts := lokiStatefulSetInstance.Spec.Template.Spec.Containers[0].VolumeMounts
	lokiStatefulSetInstanceVolumes := lokiStatefulSetInstance.Spec.Template.Spec.Volumes

	if len(lokiStatefulSetInstanceVolumeMounts) != 1 {
		t.Errorf("MountConfigMap() error = Expected 1 volume mount, got %d", len(lokiStatefulSetInstanceVolumeMounts))
		return
	}

	expectedVolumeName := fmt.Sprintf("%s-volume", configMap.Name)

	if lokiStatefulSetInstanceVolumeMounts[0].Name != expectedVolumeName {
		t.Errorf(
			"MountConfigMap() error = Expected volume mount name to be '%s', got '%s'",
			configMap.Name,
			lokiStatefulSetInstanceVolumeMounts[0].Name,
		)
		return
	}

	if lokiStatefulSetInstanceVolumeMounts[0].MountPath != mountPath {
		t.Errorf(
			"MountConfigMap() error = Expected volume mount path to be '%s', got '%s'",
			mountPath,
			lokiStatefulSetInstanceVolumeMounts[0].MountPath,
		)
		return
	}

	if len(lokiStatefulSetInstanceVolumes) != 1 {
		t.Errorf("MountConfigMap() error = Expected 1 volume, got %d", len(lokiStatefulSetInstanceVolumes))
		return
	}

	if lokiStatefulSetInstanceVolumes[0].Name != expectedVolumeName {
		t.Errorf(
			"MountConfigMap() error = Expected volume name to be '%s', got '%s'",
			configMap.Name,
			lokiStatefulSetInstanceVolumes[0].Name,
		)
		return
	}

	if lokiStatefulSetInstanceVolumes[0].ConfigMap.Name != configMap.Name {
		t.Errorf(
			"MountConfigMap() error = Expected volume configMap name to be '%s', got '%s'",
			configMap.Name,
			lokiStatefulSetInstanceVolumes[0].ConfigMap.Name,
		)
		return
	}
}

func TestUnmountConfigMapFromStatefulSet(t *testing.T) {
	configMapName := "loki-config"
	configMapData := map[string]string{"foo": "bar"}
	configMapLabels := map[string]string{"app.kubernetes.io/name": "loki"}

	configMap, err := CreateOrUpdateConfigMap(k8sClient, NAMESPACE, configMapName, configMapData, configMapLabels, Options{})
	if err != nil {
		t.Errorf("UnmountConfigMapFromStatefulSet() setup error = %v", err)
		return
	}

	_, err = deleteLokiStatefulSet()
	if err != nil {
		t.Errorf("UnmountConfigMapFromStatefulSet() setup error = %v", err)
		return
	}

	lokiStatefulSetInstance, err := createLokiStatefulset()
	if err != nil {
		t.Errorf(":UnmountConfigMapFromStatefulSet() setup error = %v", err)
		return
	}

	err = MountConfigMap(k8sClient, configMap, "/var/loki", lokiStatefulSetInstance, Options{})
	if err != nil {
		t.Errorf("UnmountConfigMapFromStatefulSet() setup error = %v", err)
		return
	}

	err = UnmountConfigMapFromStatefulSet(k8sClient, configMap.Name, lokiStatefulSetInstance, Options{})
	if err != nil {
		t.Errorf("UnmountConfigMapFromStatefulSet() error = %v", err)
		return
	}

	clearSingletons()
	lokiStatefulSetInstance, err = GetLokiStatefulSetInstance(k8sClient, &metav1.LabelSelector{MatchLabels: lokiStatefulSetInstance.Labels}, NAMESPACE, Options{})
	if err != nil {
		t.Errorf("UnmountConfigMapFromStatefulSet() setup error = %v", err)
		return
	}

	lokiStatefulSetInstanceVolumeMounts := lokiStatefulSetInstance.Spec.Template.Spec.Containers[0].VolumeMounts
	lokiStatefulSetInstanceVolumes := lokiStatefulSetInstance.Spec.Template.Spec.Volumes

	if len(lokiStatefulSetInstanceVolumeMounts) != 0 {
		t.Errorf("UnmountConfigMapFromStatefulSet() error = Expected 0 volume mounts, got %d", len(lokiStatefulSetInstanceVolumeMounts))
		return
	}

	if len(lokiStatefulSetInstanceVolumes) != 0 {
		t.Errorf("UnmountConfigMapFromStatefulSet() error = Expected 0 volumes, got %d", len(lokiStatefulSetInstanceVolumes))
		return
	}
}

func clearSingletons() {
	LOKI_STATEFUL_SET_INSTANCE = nil
}

func createLokiStatefulset() (*appsv1.StatefulSet, error) {
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

func deleteLokiStatefulSet() (noop bool, returnErr error) {
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
