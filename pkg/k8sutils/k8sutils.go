package k8sutils

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/go-kit/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	Logger log.Logger
	Ctx    context.Context
}

func sanitizeOptions(args Options) Options {
	if args.Logger == nil {
		args.Logger = log.NewNopLogger()
	}

	if args.Ctx == nil {
		args.Ctx = context.TODO()
	}

	return args
}

func getDeployments(cli client.Client, labelSelector metav1.LabelSelector, args Options) (*appsv1.DeploymentList, error) {
	ctx, logger := args.Ctx, args.Logger

	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		logger.Log("msg", "failed to convert label selector to selector", "err", err)
		return nil, err
	}

	deployments := &appsv1.DeploymentList{}

	err = cli.List(ctx, deployments, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     "default",
	})

	if err != nil {
		logger.Log("msg", "failed to list deployments", "err", err)
		return nil, err
	}

	return deployments, nil
}

func genVolumeNameFromConfconfigMap(confconfigMap *corev1.ConfigMap) string {
	return fmt.Sprintf("%s-volume", confconfigMap.Name)
}

func genAnnotationNameFromConfconfigMap(confconfigMap *corev1.ConfigMap) string {
	return fmt.Sprintf("checksum/config-%s", confconfigMap.Name)
}

func hashConfconfigMapData(confconfigMap *corev1.ConfigMap) (string, error) {
	data, err := json.Marshal(confconfigMap.Data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func volumeExists(volumeName string, deployment appsv1.Deployment) bool {
	for _, v := range deployment.Spec.Template.Spec.Volumes {
		if v.Name == volumeName {
			return true
		}
	}
	return false
}

func volumeIsMounted(volumeName string, deployment appsv1.Deployment) bool {
	for _, vm := range deployment.Spec.Template.Spec.Containers[0].VolumeMounts {
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

// CreateOrUpdateConfconfigMap creates or updates a ConfconfigMap in a specific namespace.
func CreateOrUpdateConfconfigMap(
	cli client.Client,
	namespace string,
	confconfigMap *corev1.ConfigMap,
	args Options,
) error {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	found := &corev1.ConfigMap{}
	err := cli.Get(ctx, types.NamespacedName{
		Name:      confconfigMap.Name,
		Namespace: namespace,
	}, found)

	if errors.IsNotFound(err) {
		log.Log("msg", "Creating a new ConfconfigMap", "ConfconfigMap.Namespace", namespace, "ConfconfigMap.Name", confconfigMap.Name)
		return cli.Create(ctx, confconfigMap)
	} else if err != nil {
		return err
	}

	log.Log("msg", "Updating ConfconfigMap", "ConfconfigMap.Namespace", namespace, "ConfconfigMap.Name", confconfigMap.Name)
	return cli.Update(ctx, confconfigMap)
}

// AttachConfconfigMapToDeployments attaches a ConfconfigMap to a Deployment if a volume with the matching generated
// name does not already exist in the Deployment.
func MountConfigMapToDeployments(
	cli client.Client,
	labelSelector metav1.LabelSelector,
	namespace string,
	mountPath string,
	confconfigMap *corev1.ConfigMap,
	args Options,
) error {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	deployments, err := getDeployments(cli, labelSelector, args)

	if err != nil {
		return err
	}

	if len(deployments.Items) == 0 {
		log.Log("msg", "no deployments found")
		return nil
	}

	volumeName := genVolumeNameFromConfconfigMap(confconfigMap)

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: confconfigMap.Name,
				},
			},
		},
	}

	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	}

	configMapAnnotationName := genAnnotationNameFromConfconfigMap(confconfigMap)
	configMapHash, err := hashConfconfigMapData(confconfigMap)
	if err != nil {
		log.Log("msg", "failed to hash confconfigMap data", "err", err)
		return err
	}

	for _, deployment := range deployments.Items {
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}

		deployment.Spec.Template.Annotations[configMapAnnotationName] = configMapHash

		if !volumeExists(volume.Name, deployment) && !volumeIsMounted(volume.Name, deployment) {
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)

			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
				deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
				volumeMount,
			)
		}

		err = cli.Patch(ctx, &deployment, client.Merge)
		if err != nil {
			log.Log("msg", "failed to patch deployment", "deployment", deployment.Name, "err", err)
			return err
		}
	}

	return nil
}

// DetachConfconfigMapFromDeployments detaches a ConfconfigMap from a Deployment if a volume with the matching generated
// name exists in the Deployment.
func UnmountConfigMapFromDeployments(
	cli client.Client,
	confconfigMap *corev1.ConfigMap,
	labelSelector metav1.LabelSelector,
	namespace string,
	args Options,
) error {
	args = sanitizeOptions(args)
	ctx, log := args.Ctx, args.Logger

	deployments, err := getDeployments(cli, labelSelector, args)
	if err != nil {
		return err
	}

	if len(deployments.Items) == 0 {
		log.Log("msg", "no deployments found")
		return nil
	}

	volumeName := genVolumeNameFromConfconfigMap(confconfigMap)
	confconfigMapAnnotationName := genAnnotationNameFromConfconfigMap(confconfigMap)

	for _, deployment := range deployments.Items {
		if !volumeExists(volumeName, deployment) && !volumeIsMounted(volumeName, deployment) {
			log.Log("msg", "volume does not exist in deployment", "deployment", deployment.Name)
			continue
		}

		deployment.Spec.Template.Spec.Volumes = removeVolumeByName(deployment.Spec.Template.Spec.Volumes, volumeName)
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = removeVolumeMountByName(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
			volumeName,
		)

		delete(deployment.Spec.Template.Annotations, confconfigMapAnnotationName)

		err = cli.Update(ctx, &deployment)
		if err != nil {
			log.Log("msg", "failed to update deployment", "deployment", deployment.Name, "err", err)
			return err
		}
	}

	return nil
}
