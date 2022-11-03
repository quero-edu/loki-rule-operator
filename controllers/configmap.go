package controllers

import (
	"context"
	"fmt"

	mydomainv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ensureConfigmap ensures Configmap exists in a namespace.
func (r *TravellerReconciler) ensureConfigmap(
	request reconcile.Request,
	instance *mydomainv1alpha1.Traveller,
	configmap *corev1.ConfigMap,
) error {

	found := &corev1.ConfigMap{}
	err := r.Get(context.TODO(), types.NamespacedName{
		Name:      configmap.Name,
		Namespace: instance.Namespace,
	}, found)

	if err != nil && errors.IsNotFound(err) {
		return r.Create(context.TODO(), configmap)
	} else if err != nil {
		return err
	}
	return nil
}

// configMap is a code for creating a ConfigMap
func (r *TravellerReconciler) configMap(traveler *mydomainv1alpha1.Traveller) *corev1.ConfigMap {
	labels := labels(traveler)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      traveler.Spec.Name,
			Namespace: traveler.Namespace,
			Labels:    labels,
		},
		Data: traveler.Spec.Data,
	}

	controllerutil.SetControllerReference(traveler, configMap, r.Scheme)
	return configMap
}

func (r *TravellerReconciler) syncConfigMap(configMap *corev1.ConfigMap) error {
	err := r.Update(context.TODO(), configMap)
	return err
}

func (r *TravellerReconciler) ensureConfigMapIsAttached(
	traveler *mydomainv1alpha1.Traveller,
	configMap *corev1.ConfigMap,
) error {
	deployments, err := r.getTravellerTargetDeployments(traveler)

	if err != nil {
		return err
	}

	if len(deployments.Items) == 0 {
		return fmt.Errorf("no deployments found for traveller %s", traveler.Name)
	}

	volumeName := fmt.Sprintf("%s-volume", configMap.Name)

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMap.Name,
				},
			},
		},
	}

	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: traveler.Spec.MountPath,
	}

	for _, deployment := range deployments.Items {
		if volumeExists(volume, deployment) && volumeIsMounted(volumeMount, deployment) {
			continue
		}

		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
			volumeMount,
		)

		err = r.Patch(context.TODO(), &deployment, client.Merge)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TravellerReconciler) getTravellerTargetDeployments(
	traveler *mydomainv1alpha1.Traveller,
) (*appsv1.DeploymentList, error) {
	selector, err := metav1.LabelSelectorAsSelector(&traveler.Spec.Selector)
	if err != nil {
		return nil, err
	}

	deployments := &appsv1.DeploymentList{}

	err = r.List(context.TODO(), deployments, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     "default",
	})

	if err != nil {
		return nil, err
	}

	return deployments, nil
}

func volumeExists(volume corev1.Volume, deployment appsv1.Deployment) bool {
	for _, v := range deployment.Spec.Template.Spec.Volumes {
		if v.Name == volume.Name {
			return true
		}
	}
	return false
}

func volumeIsMounted(volumeMount corev1.VolumeMount, deployment appsv1.Deployment) bool {
	for _, vm := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
		if vm.Name == volumeMount.Name {
			return true
		}
	}
	return false
}
