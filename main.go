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

package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/go-kit/log/level"
	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"github.com/quero-edu/loki-rule-operator/internal/log"
	"github.com/quero-edu/loki-rule-operator/pkg/controllers"
)

var logger = log.NewLogger("all")

var (
	scheme   = runtime.NewScheme()
	setupLog = struct {
		Info  func(keyvals ...interface{}) error
		Error func(keyvals ...interface{}) error
	}{
		Info:  level.Info(logger).Log,
		Error: level.Error(logger).Log,
	}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(querocomv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	var leaderElectionNamespace string
	var leaderElectionId string
	var logLevel string
	var lokiLabelSelector string
	var lokiNamespace string
	var lokiRuleMountPath string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "default", "The namespace where the leader election configmap will be created.")
	flag.StringVar(&leaderElectionId, "leader-election-id", "21ccfc3d.quero.com", "The id used to distinguish between multiple controller manager instances.")
	flag.StringVar(&logLevel, "log-level", "info", "The log level (debug, info, warn, error, all).")
	flag.StringVar(&lokiLabelSelector, "loki-label-selector", "", "The label selector used to filter loki instances.")
	flag.StringVar(&lokiNamespace, "loki-namespace", "default", "The namespace where the operator will operate (same as target loki instance).")
	flag.StringVar(&lokiRuleMountPath, "loki-rule-mount-path", "/etc/loki/rules", "The path where the operator will mount the loki rules configmap.")
	flag.Parse()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		MetricsBindAddress:            metricsAddr,
		Port:                          9443,
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              leaderElectionId,
		LeaderElectionNamespace:       leaderElectionNamespace,
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.LokiRuleReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: log.NewLogger(logLevel),
	}).SetupWithManager(mgr, lokiNamespace, lokiLabelSelector, lokiRuleMountPath); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LokiRule")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
