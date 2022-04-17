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
	"fmt"
	"reflect"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/jgomezve/aci-operator/api/v1alpha1"
	apicv1alpha1 "github.com/jgomezve/aci-operator/api/v1alpha1"
	"github.com/jgomezve/aci-operator/pkg/aci"
)

var (
	finalizersSegPol []string = []string{"finalizers.segmentationpolicies.apic.aci.cisco/delete"}
)

// SegmentationPolicyReconciler reconciles a SegmentationPolicy object
type SegmentationPolicyReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	ApicClient *aci.ApicClient
}

//+kubebuilder:rbac:groups=apic.aci.cisco,resources=segmentationpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=segmentationpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=segmentationpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=namespaces,verbs=get;list;watch;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SegmentationPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *SegmentationPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	segPolObject := &v1alpha1.SegmentationPolicy{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, segPolObject)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error occurred while fetching the Segmentation Policy resource")
		return ctrl.Result{}, err
	}

	// if the event is not related to delete, just check if the finalizers are rightfully set on the resource
	if segPolObject.GetDeletionTimestamp().IsZero() && !reflect.DeepEqual(finalizersSegPol, segPolObject.GetFinalizers()) {
		// set the finalizers of the Tenant to the rightful ones
		segPolObject.SetFinalizers(finalizersSegPol)
		if err := r.Update(ctx, segPolObject); err != nil {
			logger.Error(err, "error occurred while setting the finalizers of the Tenant resource")
			return ctrl.Result{}, err
		}
	}

	if !segPolObject.GetDeletionTimestamp().IsZero() {
		logger.Info("Deletion detected! Proceeding to cleanup the finalizers...")
		if err := r.deleteSegPolicyFinalizerCallback(ctx, logger, segPolObject); err != nil {
			logger.Error(err, "error occurred while dealing with the delete finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	namespaces := &corev1.NamespaceList{}
	r.List(ctx, namespaces)

	// r.ApicClient.CreateApplicationProfile(segPolObject.Spec.Name, "", segPolObject.Spec.Tenant)
	// logger.Info(fmt.Sprintf("Creating Application Profile %s", segPolObject.Spec.Name))
	// for _, nsCluster := range namespaces.Items {
	// 	for _, nsPol := range segPolObject.Spec.Namespaces {
	// 		if nsCluster.ObjectMeta.Name == nsPol {
	// 			logger.Info(fmt.Sprintf("Creating EPG for Namespace %s", nsPol))
	// 			r.ApicClient.CreateEndpointGroup(nsPol, "K8s Operator", segPolObject.Spec.Name, segPolObject.Spec.Tenant)
	// 		}
	// 	}
	// }

	for _, rule := range segPolObject.Spec.Rules {
		eth := rule.Eth
		ip := rule.IP
		port := rule.Port
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Spec.Name, eth, ip, strconv.Itoa(port))
		r.ApicClient.CreateFilterAndFilterEntry(segPolObject.Spec.Tenant, filterName, eth, ip, port)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SegmentationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apicv1alpha1.SegmentationPolicy{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.nameSpaceSegPolicyMapFunc)).
		Complete(r)
}

func (r *SegmentationPolicyReconciler) nameSpaceSegPolicyMapFunc(object client.Object) []reconcile.Request {
	ns := object.(*corev1.Namespace)

	fmt.Printf("Namespace %s modified\n", ns.Name)

	// requests := make([]reconcile.Request, 1)
	// requests[0] = reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "testNs", Name: "testName"}}
	return []reconcile.Request{}
}

func (r *SegmentationPolicyReconciler) deleteSegPolicyFinalizerCallback(ctx context.Context, logger logr.Logger, segPolObject *v1alpha1.SegmentationPolicy) error {

	// delete the row with the above 'id' from the above 'table'

	for _, rule := range segPolObject.Spec.Rules {
		eth := rule.Eth
		ip := rule.IP
		port := rule.Port
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Spec.Name, eth, ip, strconv.Itoa(port))
		if err := r.ApicClient.DeleteFilter(segPolObject.Spec.Tenant, filterName); err != nil {
			return fmt.Errorf("error occurred while deleting filter: %w", err)
		}

	}
	// remove the cleanup-row finalizer from the postgresWriterObject
	controllerutil.RemoveFinalizer(segPolObject, "finalizers.segmentationpolicies.apic.aci.cisco/delete")
	if err := r.Update(ctx, segPolObject); err != nil {
		return fmt.Errorf("error occurred while removing the finalizer: %w", err)
	}
	logger.Info("cleaned up the 'finalizers.segmentationpolicies.apic.aci.cisco/delete' finalizer successfully")
	return nil
}
