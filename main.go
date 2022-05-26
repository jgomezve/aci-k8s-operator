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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	remotecommand "k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apicv1alpha1 "github.com/jgomezve/aci-operator/api/v1alpha1"
	"github.com/jgomezve/aci-operator/controllers"
	"github.com/jgomezve/aci-operator/pkg/aci"
	"github.com/tidwall/gjson"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	user     string
	password string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(apicv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	user, password = os.Getenv("APIC_USERNAME"), os.Getenv("APIC_PASSWORD")
}

func getApicInformation(c client.Client, r rest.Interface, rc *rest.Config, s *runtime.Scheme) (controllers.AciCniConfig, error) {

	// Read CNI Connfiguration from ConfigMap
	configMap := &corev1.ConfigMapList{}
	err := c.List(context.TODO(), configMap, client.InNamespace("aci-containers-system"), client.MatchingFields{"metadata.name": "aci-containers-config"})
	if err != nil {
		return controllers.AciCniConfig{}, fmt.Errorf(fmt.Sprintf("Error reading ConfigMap aci-containers-config %s", err))
	}
	// Get the name of the controller Pod
	pods := &corev1.PodList{}
	err = c.List(context.TODO(), pods, client.InNamespace("aci-containers-system"))
	if err != nil {
		return controllers.AciCniConfig{}, fmt.Errorf(fmt.Sprintf("Error reading Controller Pod %s", err))
	}
	podController := ""
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "controller") {
			podController = pod.Name
			break
		}
	}
	if podController == "" {
		return controllers.AciCniConfig{}, fmt.Errorf(" Controller Pod not found")
	}
	// Get the private key from the Controller Pod
	execReq := r.Post().
		Namespace("aci-containers-system").
		Resource("pods").
		Name(podController).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: []string{"/bin/sh", "-c", fmt.Sprintf("cat %s", gjson.Get(configMap.Items[0].Data["controller-config"], "apic-private-key-path").String())},
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
		}, runtime.NewParameterCodec(s))

	exec, err := remotecommand.NewSPDYExecutor(rc, "POST", execReq.URL())
	if err != nil {
		return controllers.AciCniConfig{}, fmt.Errorf(fmt.Sprintf("Error setting up remote command %s", err))
	}
	cert := new(strings.Builder)
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: cert,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		return controllers.AciCniConfig{}, fmt.Errorf(fmt.Sprintf("Error executing command on Controller Pod %s", err))
	}

	return controllers.AciCniConfig{
		ApicIp:                        gjson.Get(configMap.Items[0].Data["controller-config"], "apic-hosts.1").String(),
		ApicUsername:                  gjson.Get(configMap.Items[0].Data["controller-config"], "apic-username").String(),
		ApicPrivateKey:                cert.String(),
		KeyPath:                       gjson.Get(configMap.Items[0].Data["controller-config"], "apic-private-key-path").String(),
		PolicyTenant:                  gjson.Get(configMap.Items[0].Data["controller-config"], "aci-policy-tenant").String(),
		PodBridgeDomain:               strings.Replace(strings.Split(gjson.Get(configMap.Items[0].Data["controller-config"], "aci-podbd-dn").String(), "/")[2], "BD-", "", -1),
		KubernetesVmmDomain:           gjson.Get(configMap.Items[0].Data["controller-config"], "aci-vmm-domain").String(),
		ApplicationProfileKubeDefault: gjson.Get(configMap.Items[0].Data["host-agent-config"], "app-profile").String(),
		EPGKubeDefault:                strings.Split(gjson.Get(configMap.Items[0].Data["host-agent-config"], "default-endpoint-group.name").String(), "|")[1],
	}, nil
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
		// Disable cache for configMaps/Pods as we need to read ACI Configuration before starting the Manager/Controllers
		ClientDisableCacheFor: []client.Object{&corev1.ConfigMap{}, &corev1.Pod{}},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	gvk := schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}

	restClient, _ := apiutil.RESTClientForGVK(gvk, false, mgr.GetConfig(), serializer.NewCodecFactory(mgr.GetScheme()))
	cniConf, err := getApicInformation(mgr.GetClient(), restClient, mgr.GetConfig(), mgr.GetScheme())
	if err != nil {
		setupLog.Error(err, "unable to read ACI CNI configuration")
	}
	setupLog.Info(fmt.Sprintf("ACI CNI configuration discovered for tenant %s in APIC controller %s", cniConf.PolicyTenant, cniConf.ApicIp))

	apicClient, err := aci.NewApicClient(cniConf.ApicIp, cniConf.ApicUsername, password, cniConf.ApicPrivateKey)
	if err != nil {
		setupLog.Error(err, "unable to setup the Apic Client")
		os.Exit(1)
	}

	if err = (&controllers.SegmentationPolicyReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		ApicClient: apicClient,
		CniConfig:  cniConf,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SegmentationPolicy")
		os.Exit(1)
	}

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
