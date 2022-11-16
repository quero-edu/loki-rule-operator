/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"github.com/quero-edu/loki-rule-operator/pkg/k8sutils"
	"github.com/quero-edu/loki-rule-operator/pkg/lokirule"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// LokiRuleReconciler reconciles a LokiRule object
type LokiRuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger log.Logger
}

//+kubebuilder:rbac:groups=quero.com,resources=lokiRules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=quero.com,resources=lokiRules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=quero.com,resources=lokiRules/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LokiRule object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *LokiRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	level.Info(r.Logger).Log("msg", "Reconciling LokiRule", "namespace", req.NamespacedName)

	// Fetch the LokiRule instance
	instance := &querocomv1alpha1.LokiRule{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	configMap := lokirule.GenerateConfigMap(instance)

	controllerutil.SetControllerReference(instance, configMap, r.Scheme)

	err = k8sutils.CreateOrUpdateConfigMap(
		r.Client,
		instance.Namespace,
		configMap,
		k8sutils.Options{Ctx: ctx, Logger: level.Debug(r.Logger)},
	)
	if err != nil {
		level.Error(r.Logger).Log("err", err, "msg", "Failed to ensure configMap exists")
		return reconcile.Result{}, err
	}

	err = k8sutils.MountConfigMapToDeployments(
		r.Client,
		instance.Spec.Selector,
		instance.Namespace,
		instance.Spec.MountPath,
		configMap,
		k8sutils.Options{Ctx: ctx, Logger: level.Debug(r.Logger)},
	)
	if err != nil {
		level.Error(r.Logger).Log("err", err, "msg", "ConfigMap not attached")
		return reconcile.Result{}, err
	}

	level.Info(r.Logger).Log("msg", "LokiRule Reconciled")

	return ctrl.Result{}, nil
}

func handleByEventType(r *LokiRuleReconciler) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			deletedInstance := e.Object.(*querocomv1alpha1.LokiRule)

			level.Info(r.Logger).Log("msg", "Reconciling deleted LokiRule", "namespace", deletedInstance.Namespace, "name", deletedInstance.Name)

			configMap := lokirule.GenerateConfigMap(deletedInstance)

			level.Debug(r.Logger).Log("msg", "Unmounting configMap from deployments", "configMap", configMap.Name)
			err := k8sutils.UnmountConfigMapFromDeployments(
				r.Client,
				configMap,
				deletedInstance.Spec.Selector,
				deletedInstance.Namespace,
				k8sutils.Options{Ctx: context.Background(), Logger: level.Debug(r.Logger)},
			)

			if err != nil {
				level.Error(r.Logger).Log("err", err, "msg", "Failed to unmount configMap from deployments")
			}

			level.Info(r.Logger).Log("msg", "deleted LokiRule reconciled")

			return false
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *LokiRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&querocomv1alpha1.LokiRule{}).
		WithEventFilter(handleByEventType(r)).
		Complete(r)
}
