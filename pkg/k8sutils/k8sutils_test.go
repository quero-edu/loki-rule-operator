package k8sutils

import (
	// "context"
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

	k8sClient.Delete(context.TODO(), namespace)
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
	labels := map[string]string{
		"app": "test",
	}

	deploymentName := "test-deployment"

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
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
	deployment = &appsv1.Deployment{}

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
}

func TestUnmountConfigMapFromDeployments(t *testing.T) {
	labels := map[string]string{
		"app": "test",
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
	deploymentName := "test-deployment-with-volume"

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
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
					Volumes: []corev1.Volume{
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
					},
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      fmt.Sprintf("%s-volume", configMap.Name),
									MountPath: "/etc/config",
								},
							},
						},
					},
				},
			},
		},
	}

	err := k8sClient.Create(context.TODO(), deployment)

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

}
