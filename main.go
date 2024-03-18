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
	"fmt"
	"io"
	"os"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"github.com/quero-edu/loki-rule-operator/internal/flags"
	httputil "github.com/quero-edu/loki-rule-operator/internal/http"
	"github.com/quero-edu/loki-rule-operator/internal/logger"
	"github.com/quero-edu/loki-rule-operator/pkg/controllers"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logCtrl "sigs.k8s.io/controller-runtime/pkg/log"
	metricsServer "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(querocomv1alpha1.AddToScheme(scheme))
}

func main() {
	var logErrorCallback = func(loggerError error, args ...interface{}) {
		errorMsg := fmt.Sprintf("Error, could not log: %s, args: %v", loggerError.Error(), args)
		if _, err := io.Writer(os.Stderr).Write([]byte(errorMsg)); err != nil {
			os.Exit(1)
		}
	}

	var log = logger.NewLogger("all", logErrorCallback)
	logCtrl.SetLogger(logr.New(logCtrl.NullLogSink{}))

	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	var leaderElectionNamespace string
	var leaderElectionID string
	var logLevel string
	var lokiLabelSelector string
	var lokiNamespace string
	var lokiRuleMountPath string
	var lokiURL string
	var lokiHeaders flags.ArrayFlags
	var onlyReconcileRules bool

	flag.BoolVar(
		&enableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active manager.",
	)
	flag.StringVar(
		&metricsAddr,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to.",
	)
	flag.StringVar(
		&probeAddr,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.",
	)
	flag.StringVar(
		&leaderElectionNamespace,
		"leader-election-namespace",
		"default",
		"The namespace where the leader election configmap will be created.",
	)
	flag.StringVar(
		&leaderElectionID,
		"leader-election-id",
		"21ccfc3d.quero.com",
		"The id used to distinguish between multiple controller manager instances.",
	)
	flag.StringVar(
		&logLevel,
		"log-level",
		"info",
		"The log level (debug, info, warn, error, all).",
	)
	flag.StringVar(
		&lokiLabelSelector,
		"loki-label-selector",
		"",
		"The label selector used to filter loki instances.",
	)
	flag.StringVar(
		&lokiNamespace,
		"loki-namespace",
		"default",
		"The namespace where the operator will operate (same as target loki instance).",
	)
	flag.StringVar(
		&lokiRuleMountPath,
		"loki-rule-mount-path",
		"/etc/loki/rules",
		"The path where the operator will mount the loki rules configmap.",
	)
	flag.StringVar(
		&lokiURL,
		"loki-url",
		"",
		"Loki server URL.",
	)
	flag.Var(
		&lokiHeaders,
		"loki-header",
		"Extra header that will be sent to Loki. Format KEY=VALUE. May be repeated.",
	)
	flag.BoolVar(
		&onlyReconcileRules,
		"only-reconcile-rules",
		false,
		"When enabled the operator will only reconcile LokiRule's into the ConfigMap. "+
			"It will skip updating the DaemonSet volume, volumeMounts and annotation hash, "+
			"efficiently avoiding restarts of Loki.",
	)

	flag.Parse()

	metricsServerOpts := metricsServer.Options{
		BindAddress: metricsAddr,
	}

	webhookServer := webhook.NewServer(webhook.Options{
		Port: 9443,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		Metrics:                       metricsServerOpts,
		WebhookServer:                 webhookServer,
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              leaderElectionID,
		LeaderElectionNamespace:       leaderElectionNamespace,
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	lokiSelector, err := metav1.ParseToLabelSelector(lokiLabelSelector)
	if err != nil {
		log.Error(err, "unable to parse loki label selector")
		os.Exit(1)
	}

	if lokiNamespace == "" {
		lokiNamespace = "default"
	}

	if err = (&controllers.LokiRuleReconciler{
		Client:                mgr.GetClient(),
		Scheme:                mgr.GetScheme(),
		Logger:                log,
		LokiClient:            httputil.ClientWithHeaders(&lokiHeaders),
		LokiRulesPath:         lokiRuleMountPath,
		LokiLabelSelector:     lokiSelector,
		LokiNamespace:         lokiNamespace,
		LokiRuleConfigMapName: "loki-rule-cfg",
		LokiURL:               lokiURL,
		UpdateLoki:            !onlyReconcileRules,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "LokiRule")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	log.Info("starting manager", "onlyReconcileRules", onlyReconcileRules)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
