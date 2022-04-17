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

	"github.com/jgomezve/aci-operator/pkg/aci"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	"github.com/jgomezve/aci-operator/api/v1alpha1"
	apicv1alpha1 "github.com/jgomezve/aci-operator/api/v1alpha1"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	ApicClient aci.ApicInterface
}

//+kubebuilder:rbac:groups=apic.aci.cisco,resources=tenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=tenants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=tenants/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tenant object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	tenantObject := &v1alpha1.Tenant{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, tenantObject)
	if err != nil {
		// if the resource is not found, then just return (might look useless as this usually happens in case of Delete events)
		if errors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("Tenant %s in Namespace %s not found", req.Name, req.Namespace))
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error occurred while fetching the Tenant resource")
		return ctrl.Result{}, err
	}

	// if the event is not related to delete, just check if the finalizers are rightfully set on the resource
	if tenantObject.GetDeletionTimestamp().IsZero() && !reflect.DeepEqual(finalizers, tenantObject.GetFinalizers()) {
		// set the finalizers of the Tenant to the rightful ones
		tenantObject.SetFinalizers(finalizers)
		if err := r.Update(ctx, tenantObject); err != nil {
			logger.Error(err, "error occurred while setting the finalizers of the Tenant resource")
			return ctrl.Result{}, err
		}
	}

	// if the metadata.deletionTimestamp is found to be non-zero, this means that the resource is intended and just about to be deleted
	// hence, it's time to clean up the finalizers
	if !tenantObject.GetDeletionTimestamp().IsZero() {
		logger.Info("Deletion detected! Proceeding to cleanup the finalizers...")
		if err := r.deleteTenantFinalizerCallback(ctx, logger, tenantObject); err != nil {
			logger.Error(err, "error occurred while dealing with the delete finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	name, description := tenantObject.Spec.Name, tenantObject.Spec.Description
	logger.Info(fmt.Sprintf("Creating Tenant %s (%s)", name, description))

	if err = r.ApicClient.CreateTenant(name, description); err != nil {
		logger.Error(err, "error occurred while Creating Tenant")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apicv1alpha1.Tenant{}).
		Complete(r)
}

func (r *TenantReconciler) deleteTenantFinalizerCallback(ctx context.Context, logger logr.Logger, tenantObject *v1alpha1.Tenant) error {

	// delete the row with the above 'id' from the above 'table'
	if err := r.ApicClient.DeleteTenant(tenantObject.Spec.Name); err != nil {
		return fmt.Errorf("error occurred while deleting Tenant: %w", err)
	}

	// remove the cleanup-row finalizer from the postgresWriterObject
	controllerutil.RemoveFinalizer(tenantObject, "finalizers.tenants.apic.aci.cisco/delete")
	if err := r.Update(ctx, tenantObject); err != nil {
		return fmt.Errorf("error occurred while removing the finalizer: %w", err)
	}
	logger.Info("cleaned up the 'finalizers.tenants.apic.aci.cisco/delete' finalizer successfully")
	return nil
}
