package k8sutils

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	configMapName := "test-configmap"
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: NAMESPACE,
		},
	}

	// Create a configmap
	err := CreateOrUpdateConfigmap(k8sClient, NAMESPACE, configMap, Options{})
	if err != nil {
		t.Errorf("CreateOrUpdateConfigmap() error = %v", err)
		return
	}

	// Asserts that the configmap was created
	configMap = &corev1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      configMapName,
		Namespace: NAMESPACE,
	}, configMap)

	if err != nil {
		t.Errorf("CreateOrUpdateConfigmap() error = %v", err)
		return
	}

	if configMap.Name != configMapName {
		t.Errorf(
			"CreateOrUpdateConfigmap() error = ConfigMap was not created. Expected name to be '%s', got '%s'",
			configMapName,
			configMap.Name,
		)
		return
	}

	// Update a configmap
	configMap.Data = map[string]string{"foo": "bar"}
	err = CreateOrUpdateConfigmap(k8sClient, NAMESPACE, configMap, Options{})
	if err != nil {
		t.Errorf("CreateOrUpdateConfigmap() error = %v", err)
		return
	}

	// Asserts that the configmap was updated
	configMap = &corev1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      configMapName,
		Namespace: NAMESPACE,
	}, configMap)

	if err != nil {
		t.Errorf("CreateOrUpdateConfigmap() error = %v", err)
		return
	}

	if configMap.Data["foo"] != "bar" {
		t.Errorf(
			"CreateOrUpdateConfigmap() error = Configmap was not updated, expected configMap.Data[\"foo\"] to be 'bar',got '%s'",
			configMap.Data["foo"],
		)
		return
	}
}

func TestMountConfigMapToDeployments(t *testing.T) {
	deploymentName := "test-deployment"
	labels := map[string]string{
		"test": "mountCfgMap",
	}

	_, err := createSimpleDeployment(deploymentName, labels)
	if err != nil {
		t.Errorf("MountConfigMapToDeployments() setup error = %v", err)
		return
	}

	mountPath := "/etc/config"
	labelSelector := metav1.LabelSelector{
		MatchLabels: labels,
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: NAMESPACE,
		},
		Data: map[string]string{"test": "test"},
	}

	err = MountConfigMapToDeployments(
		k8sClient,
		labelSelector,
		NAMESPACE,
		mountPath,
		&configMap,
		Options{},
	)

	if err != nil {
		t.Errorf("MountConfigMapToDeployments() error = %v", err)
		return
	}

	// Asserts that the configmap was mounted to the deployment
	deployment := &appsv1.Deployment{}

	err = k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      deploymentName,
		Namespace: NAMESPACE,
	}, deployment)

	if err != nil {
		t.Errorf("MountConfigMapToDeployments() error = %v", err)
		return
	}

	if deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name != configMap.Name {
		t.Errorf(
			"MountConfigMapToDeployments() error = ConfigMap was not mounted to the deployment. Expected name to be '%s', got '%s'",
			configMap.Name,
			deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name,
		)
		return
	}

	// "{\"test\":\"test\"}" | sha256sum
	configmapDataHash := "3e80b3778b3b03766e7be993131c0af2ad05630c5d96fb7fa132d05b77336e04"
	configmapHashAnnotation := fmt.Sprintf("checksum/config-%s", configMap.Name)

	if deployment.Spec.Template.Annotations[configmapHashAnnotation] != configmapDataHash {
		t.Errorf(
			"AnnotateDeploymentWithConfigmapHash() error = Deployment was not annotated with the configmap hash. Expected annotation to be '%s', got '%s'",
			configmapDataHash,
			deployment.Spec.Template.Annotations,
		)
		return
	}
}

func TestUnmountConfigMapFromDeployments(t *testing.T) {
	labels := map[string]string{
		"app": "unmountCfgMap",
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: labels,
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: NAMESPACE,
		},
	}

	deploymentName := "test-deployment-with-volumes"

	deployment, err := createSimpleDeployment(deploymentName, labels)
	if err != nil {
		t.Errorf("UnmountConfigMapFromDeployments() setup error = %v", err)
		return
	}

	deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: fmt.Sprintf("%s-volume", configMap.Name),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMap.Name,
					},
				},
			},
		},
	}

	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			Name:      fmt.Sprintf("%s-volume", configMap.Name),
			MountPath: "/etc/config",
		},
	}

	err = k8sClient.Update(context.TODO(), deployment)

	if err != nil {
		t.Errorf("UnmountConfigMapFromDeployments() setup error = %v", err)
		return
	}

	if err != nil {
		t.Errorf("UnmountConfigMapFromDeployments() setup error = %v", err)
		return
	}

	err = UnmountConfigMapFromDeployments(
		k8sClient,
		configMap,
		labelSelector,
		NAMESPACE,
		Options{},
	)
	if err != nil {
		t.Errorf("UnmountConfigMapFromDeployments() error = %v", err)
		return
	}

	// Asserts that the configmap was unmounted from the deployment
	deployment = &appsv1.Deployment{}
	err = k8sClient.Get(context.TODO(), types.NamespacedName{
		Name:      deploymentName,
		Namespace: NAMESPACE,
	}, deployment)
	if err != nil {
		t.Errorf("UnmountConfigMapFromDeployments() error = %v", err)
		return
	}

	if len(deployment.Spec.Template.Spec.Volumes) != 0 {
		t.Errorf(
			"UnmountConfigMapFromDeployments() error = ConfigMap was not unmounted from the deployment. Expected volumes to be empty, got '%v'",
			deployment.Spec.Template.Spec.Volumes,
		)
		return
	}

	configMapHashAnnotation := fmt.Sprintf("checksum/config-%s", configMap.Name)
	if deployment.Annotations[configMapHashAnnotation] != "" {
		t.Errorf(
			"UnmountConfigMapFromDeployments() error = Deployment was not unannotated with the configmap hash. Expected annotation to be empty, got '%s'",
			deployment.Annotations[configMapHashAnnotation],
		)
		return
	}
}

func createSimpleDeployment(name string, labels map[string]string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
				},
			},
		},
	}

	err := k8sClient.Create(context.TODO(), deployment)

	if err != nil {
		return nil, err
	}

	return deployment, nil
}
