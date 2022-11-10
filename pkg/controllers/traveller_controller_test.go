package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mydomainv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"github.com/quero-edu/loki-rule-operator/internal/log"
	//+kubebuilder:scaffold:imports
)

var k8sClient client.Client
var testEnv *envtest.Environment

var travellerReconciler *TravellerReconciler

const NAMESPACE = "default"

func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic(err)
	}

	err = mydomainv1alpha1.AddToScheme(scheme.Scheme)
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

	travellerReconciler = &TravellerReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
		Logger: log.NewLogger("all"),
	}

	m.Run()

	testEnv.Stop()
}

func TestReconcile(t *testing.T) {
	const configMapName = "test-traveller-config"
	const deploymentName = "test-traveller-deployment"
	const mountPath = "/etc/traveller"

	labels := map[string]string{
		"app": "test",
	}

	traveller := &mydomainv1alpha1.Traveller{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-traveller",
			Namespace: NAMESPACE,
		},
		Spec: mydomainv1alpha1.TravellerSpec{
			Name: configMapName,
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			MountPath: mountPath,
			Data: map[string]string{
				"test": "test",
			},
		},
	}

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
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			},
		},
	}

	err := k8sClient.Create(context.TODO(), deployment)
	if err != nil {
		t.Errorf("TestReconcile() Error creating deployment: %v", err)
		return
	}

	err = k8sClient.Create(context.TODO(), traveller)
	if err != nil {
		t.Errorf("TestReconcile() Error creating traveller: %v", err)
		return
	}

	travellerReconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: client.ObjectKey{
			Name:      traveller.Name,
			Namespace: traveller.Namespace,
		},
	})

	configmap := &corev1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{
		Name:      configMapName,
		Namespace: NAMESPACE,
	}, configmap)

	if err != nil {
		t.Errorf("TestReconcile() Error getting configmap: %v", err)
		return
	}

	if configmap.Data["test"] != "test" {
		t.Errorf("TestReconcile() Assertion failed: Configmap data is not equal to traveller data")
		return
	}

	deployment = &appsv1.Deployment{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{
		Name:      deploymentName,
		Namespace: NAMESPACE,
	}, deployment)
	if err != nil {
		t.Errorf("TestReconcile() Error getting deployment: %v", err)
		return
	}

	expectedVolumeName := fmt.Sprintf("%s-volume", configMapName)

	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Errorf("TestReconcile() Assertion failed: Deployment volumes length is not equal to 1")
		return
	}

	if deployment.Spec.Template.Spec.Volumes[0].Name != expectedVolumeName {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume name is not equal to configmap name")
		return
	}

	if len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts) != 1 {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume mounts length is not equal to 1")
		return
	}

	if deployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name != expectedVolumeName {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume mount name is not equal to configmap name")
		return
	}

	if deployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath != mountPath {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume mount path is not equal to traveller mount path")
		return
	}
}
