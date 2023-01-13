package k8sutils

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/quero-edu/loki-rule-operator/internal/logger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	Logger logger.Logger
	Ctx    context.Context
}

func sanitizeOptions(args Options) Options {
	if args.Logger == nil {
		args.Logger = logger.NewNopLogger()
	}

	if args.Ctx == nil {
		args.Ctx = context.TODO()
	}

	return args
}

func genVolumeNameFromConfigMap(configMapName string) string {
	return fmt.Sprintf("%s-volume", configMapName)
}

func genHashAnnotation(configMapName string) string {
	return fmt.Sprintf("checksum/config-%s", configMapName)
}

func hashConfigMapData(configMap *corev1.ConfigMap) (string, error) {
	data, err := json.Marshal(configMap.Data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func volumeExists(volumeName string, lokiStatefulSet *appsv1.StatefulSet) bool {
	for _, v := range lokiStatefulSet.Spec.Template.Spec.Volumes {
		if v.Name == volumeName {
			return true
		}
	}
	return false
}

func volumeIsMounted(volumeName string, lokiStatefulSet *appsv1.StatefulSet) bool {
	for _, vm := range lokiStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
		if vm.Name == volumeName {
			return true
		}
	}
	return false
}

func removeVolumeByName(volumes []corev1.Volume, name string) []corev1.Volume {
	for i, volume := range volumes {
		if volume.Name == name {
			return append(volumes[:i], volumes[i+1:]...)
		}
	}

	return volumes
}

func removeVolumeMountByName(volumeMounts []corev1.VolumeMount, name string) []corev1.VolumeMount {
	for i, volumeMount := range volumeMounts {
		if volumeMount.Name == name {
			volumeMounts = append(volumeMounts[:i], volumeMounts[i+1:]...)
		}
	}

	return volumeMounts
}

func generateVolumeMounts(
	mountPath string,
	configMapName string,
) (corev1.Volume, corev1.VolumeMount) {
	volumeName := genVolumeNameFromConfigMap(configMapName)

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}

	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	}

	return volume, volumeMount
}

func GetStatefulSet(
	cli client.Client,
	labelSelector *metav1.LabelSelector,
	namespace string,
	args Options,
) (*appsv1.StatefulSet, error) {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		log.Debug("failed to convert label selector to selector", "err", err)
		return nil, err
	}

	statefulSets := &appsv1.StatefulSetList{}

	err = cli.List(ctx, statefulSets, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     namespace,
	})

	if err != nil {
		log.Debug("failed to list statefulSets", "err", err)
		return nil, err
	}

	if len(statefulSets.Items) > 1 {
		log.Debug("more than one statefulSet found")
		return nil, fmt.Errorf("more than one statefulSet found")
	}

	if len(statefulSets.Items) == 0 {
		log.Debug("no statefulSets found")
		return nil, fmt.Errorf("no statefulSets found")
	}

	return &statefulSets.Items[0], nil
}

func CreateOrUpdateConfigMap(
	cli client.Client,
	namespace string,
	configMapName string,
	configMapData map[string]string,
	labels map[string]string,
	args Options,
) (*corev1.ConfigMap, error) {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	configMap := &corev1.ConfigMap{}
	err := cli.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: namespace,
	}, configMap)

	configMap.Name = configMapName
	configMap.Namespace = namespace
	configMap.Data = configMapData
	configMap.Labels = labels

	if errors.IsNotFound(err) {
		log.Debug("Creating a new ConfigMap", "ConfigMap.Namespace", namespace, "ConfigMap.Name", configMapName)
		return configMap, cli.Create(ctx, configMap)
	} else if err != nil {
		return nil, err
	}

	log.Debug("Updating ConfigMap", "ConfigMap.Namespace", namespace, "ConfigMap.Name", configMapName)
	return configMap, cli.Update(ctx, configMap)
}

func DeleteConfigMap(cli client.Client, configMapName string, configMapNameSpace string, args Options) error {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	configMap := &corev1.ConfigMap{}
	err := cli.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: configMapNameSpace,
	}, configMap)

	if err != nil {
		log.Debug("failed to get configmap", "err", err)
		return err
	}

	log.Debug("Deleting ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
	return cli.Delete(ctx, configMap)
}

func MountConfigMap(
	cli client.Client,
	configMap *corev1.ConfigMap,
	mountPath string,
	lokiStatefulSet *appsv1.StatefulSet,
	args Options,
) error {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	volume, volumeMount := generateVolumeMounts(mountPath, configMap.Name)

	configMapAnnotationName := genHashAnnotation(configMap.Name)
	configMapHash, err := hashConfigMapData(configMap)

	if err != nil {
		log.Debug("failed to hash configmap data", "err", err)
		return err
	}

	if lokiStatefulSet.Spec.Template.Annotations == nil {
		lokiStatefulSet.Spec.Template.Annotations = make(map[string]string)
	}

	lokiStatefulSet.Spec.Template.Annotations[configMapAnnotationName] = configMapHash

	if !volumeExists(volume.Name, lokiStatefulSet) && !volumeIsMounted(volume.Name, lokiStatefulSet) {
		lokiStatefulSet.Spec.Template.Spec.Volumes = append(lokiStatefulSet.Spec.Template.Spec.Volumes, volume)

		lokiStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			lokiStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts,
			volumeMount,
		)
	}

	err = cli.Patch(ctx, lokiStatefulSet, client.Merge)
	if err != nil {
		log.Debug("failed to patch statefulSet", "statefulSet", lokiStatefulSet.Name, "err", err)
		return err
	}

	return nil
}

func UnmountConfigMap(
	cli client.Client,
	configMapName string,
	statefulSet *appsv1.StatefulSet,
	args Options,
) error {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	volumeName := genVolumeNameFromConfigMap(configMapName)
	configMapAnnotationName := genHashAnnotation(configMapName)

	if !volumeExists(volumeName, statefulSet) && !volumeIsMounted(volumeName, statefulSet) {
		log.Debug("volume does not exist in statefulSet", "statefulSet", statefulSet.Name)
		return nil
	}

	statefulSet.Spec.Template.Spec.Volumes = removeVolumeByName(statefulSet.Spec.Template.Spec.Volumes, volumeName)
	statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts = removeVolumeMountByName(
		statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts,
		volumeName,
	)

	delete(statefulSet.Spec.Template.Annotations, configMapAnnotationName)

	err := cli.Update(ctx, statefulSet)
	if err != nil {
		log.Debug("failed to update statefulSet", "statefulSet", statefulSet.Name, "err", err)
		return err
	}

	return nil
}
