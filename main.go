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
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apicv1alpha1 "github.com/jgomezve/aci-operator/api/v1alpha1"
	"github.com/jgomezve/aci-operator/controllers"
	"github.com/tidwall/gjson"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	host     string
	user     string
	password string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(apicv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	host, user, password = os.Getenv("APIC_HOST"), os.Getenv("APIC_USERNAME"), os.Getenv("APIC_PASSWORD")
}

func getApicInformation(c client.Client) controllers.AciCniConfig {

	configMap := &corev1.ConfigMapList{}
	err := c.List(context.TODO(), configMap, client.InNamespace("aci-containers-system"), client.MatchingFields{"metadata.name": "aci-containers-config"})
	if err != nil {
		fmt.Printf("Error %s", err)
	}
	return controllers.AciCniConfig{
		ApicIp:                        gjson.Get(configMap.Items[0].Data["controller-config"], "apic-hosts.1").String(),
		ApicUsername:                  gjson.Get(configMap.Items[0].Data["controller-config"], "apic-username").String(),
		KeyPath:                       gjson.Get(configMap.Items[0].Data["controller-config"], "apic-private-key-path").String(),
		PodBridgeDomain:               strings.Replace(strings.Split(gjson.Get(configMap.Items[0].Data["controller-config"], "aci-podbd-dn").String(), "/")[2], "BD-", "", -1),
		KubernetesVmmDomain:           gjson.Get(configMap.Items[0].Data["controller-config"], "aci-vmm-domain").String(),
		ApplicationProfileKubeDefault: gjson.Get(configMap.Items[0].Data["host-agent-config"], "app-profile").String(),
		EPGKubeDefault:                strings.Split(gjson.Get(configMap.Items[0].Data["host-agent-config"], "default-endpoint-group.name").String(), "|")[1],
	}
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		Port:                   9443,
		HealthProbeBindAddress: "0",
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "1d45f356.aci.cisco",
		// Disabel cache for configMaps as we need to read ACI COnfiguration before starting the Manager/Controllers
		ClientDisableCacheFor: []client.Object{&corev1.ConfigMap{}},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	fmt.Printf("%s\n", getApicInformation(mgr.GetClient()))
	// apicClient, err := aci.NewApicClient(host, user, password)
	// if err != nil {
	// 	setupLog.Error(err, "unable to setup the Apic Client")
	// 	os.Exit(1)
	// }

	// if err = (&controllers.TenantReconciler{
	// 	Client:     mgr.GetClient(),
	// 	Scheme:     mgr.GetScheme(),
	// 	ApicClient: apicClient,
	// }).SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "Tenant")
	// 	os.Exit(1)
	// }
	// if err = (&controllers.ApplicationProfileReconciler{
	// 	Client:     mgr.GetClient(),
	// 	Scheme:     mgr.GetScheme(),
	// 	ApicClient: apicClient,
	// }).SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "ApplicationProfile")
	// 	os.Exit(1)
	// }
	// if err = (&controllers.SegmentationPolicyReconciler{
	// 	Client:     mgr.GetClient(),
	// 	Scheme:     mgr.GetScheme(),
	// 	ApicClient: apicClient,
	// }).SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "SegmentationPolicy")
	// 	os.Exit(1)
	// }

	//+kubebuilder:scaffold:builder

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
