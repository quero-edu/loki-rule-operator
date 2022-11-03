package controllers

import (
	"context"

	mydomainv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"golang.org/x/exp/maps"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ensureConfigmap ensures Configmap exists in a namespace.
func (r *TravellerReconciler) ensureConfigmap(request reconcile.Request,
	instance *mydomainv1alpha1.Traveller,
	configmap *corev1.ConfigMap,
) (*reconcile.Result, error) {

	found := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Name:      configmap.Name,
		Namespace: instance.Namespace,
	}, found)

	if err != nil && errors.IsNotFound(err) {
		err = r.Create(context.TODO(), configmap)
		if err != nil {
			return &reconcile.Result{}, err
		} else {
			return nil, nil
		}
	} else if err != nil {
		return &reconcile.Result{}, err
	}

	return nil, nil
}

// configMap is a code for creating a ConfigMap
func (r *TravellerReconciler) configMap(v *mydomainv1alpha1.Traveller) *corev1.ConfigMap {
	labels := labels(v, "backend")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend-service",
			Namespace: v.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"backend-service.yaml": `test: true`,
		},
	}

	controllerutil.SetControllerReference(v, configMap, r.Scheme)
	return configMap
}

func (r *TravellerReconciler) updateConfigMap(configMap *corev1.ConfigMap, newConfigs map[string]string) error {
	maps.Copy(configMap.Data, newConfigs)
	err := r.Patch(context.TODO(), configMap, client.Merge)
	return err
}
