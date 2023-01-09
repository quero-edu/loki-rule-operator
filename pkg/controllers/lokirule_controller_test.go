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

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"github.com/quero-edu/loki-rule-operator/internal/log"
	//+kubebuilder:scaffold:imports
)

var k8sClient client.Client
var testEnv *envtest.Environment

var lokiRuleReconciler *LokiRuleReconciler

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

	err = querocomv1alpha1.AddToScheme(scheme.Scheme)
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

	lokiRuleReconciler = &LokiRuleReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
		Logger: log.NewLogger("all"),
	}

	setLokiInstanceConfig("app=loki", NAMESPACE, "/var/loki/rules")

	m.Run()

	testEnv.Stop()
}

func TestReconcile(t *testing.T) {
	const configMapName = "test-lokirule-config"

	lokiRule := &querocomv1alpha1.LokiRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lokirule",
			Namespace: NAMESPACE,
		},
		Spec: querocomv1alpha1.LokiRuleSpec{
			Name: configMapName,
			Data: map[string]string{
				"test": "test",
			},
		},
	}

	statefulSet, err := createLokiStatefulset()
	if err != nil {
		t.Errorf("TestReconcile() Error creating statefulset: %v", err)
		return
	}

	err = k8sClient.Create(context.TODO(), lokiRule)
	if err != nil {
		t.Errorf("TestReconcile() Error creating lokiRule: %v", err)
		return
	}

	lokiRuleReconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: client.ObjectKey{
			Name:      lokiRule.Name,
			Namespace: lokiRule.Namespace,
		},
	})

	configMap := &corev1.ConfigMap{}
	err = k8sClient.Get(context.TODO(), client.ObjectKey{
		Name:      configMapName,
		Namespace: NAMESPACE,
	}, configMap)

	if err != nil {
		t.Errorf("TestReconcile() Error getting configMap: %v", err)
		return
	}

	if configMap.Data["test"] != "test" {
		t.Errorf("TestReconcile() Assertion failed: ConfigMap data is not equal to lokiRule data")
		return
	}

	err = k8sClient.Get(context.TODO(), client.ObjectKey{
		Name:      statefulSet.Name,
		Namespace: NAMESPACE,
	}, statefulSet)
	if err != nil {
		t.Errorf("TestReconcile() Error getting deployment: %v", err)
		return
	}

	expectedVolumeName := fmt.Sprintf("%s-volume", configMapName)

	if len(statefulSet.Spec.Template.Spec.Volumes) != 1 {
		t.Errorf("TestReconcile() Assertion failed: Deployment volumes length is not equal to 1")
		return
	}

	if statefulSet.Spec.Template.Spec.Volumes[0].Name != expectedVolumeName {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume name is not equal to configMap name")
		return
	}

	if len(statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts) != 1 {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume mounts length is not equal to 1")
		return
	}

	if statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name != expectedVolumeName {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume mount name is not equal to configMap name")
		return
	}

	if statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath != LOKI_INSTANCE_RULE_MOUNTPATH {
		t.Errorf("TestReconcile() Assertion failed: Deployment volume mount path is not equal to lokiRule mount path")
		return
	}
}

func createLokiStatefulset() (*appsv1.StatefulSet, error) {
	labels := map[string]string{
		"app": "loki",
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
