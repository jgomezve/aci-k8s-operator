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
	"github.com/jgomezve/aci-operator/pkg/aci"
)

var (
	finalizers []string = []string{"finalizers.applicationprofiles.apic.aci.cisco/delete"}
)

// ApplicationProfileReconciler reconciles a ApplicationProfile object
type ApplicationProfileReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	ApicClient *aci.ApicClient
}

//+kubebuilder:rbac:groups=apic.aci.cisco,resources=applicationprofiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=applicationprofiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=applicationprofiles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ApplicationProfile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ApplicationProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	appObject := &v1alpha1.ApplicationProfile{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, appObject)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error occurred while fetching the Application Profile resource")
		return ctrl.Result{}, err
	}

	// if the event is not related to delete, just check if the finalizers are rightfully set on the resource
	if appObject.GetDeletionTimestamp().IsZero() && !reflect.DeepEqual(finalizers, appObject.GetFinalizers()) {
		// set the finalizers of the Tenant to the rightful ones
		appObject.SetFinalizers(finalizers)
		if err := r.Update(ctx, appObject); err != nil {
			logger.Error(err, "error occurred while setting the finalizers of the Application Profile resource")
			return ctrl.Result{}, err
		}
	}

	// if the metadata.deletionTimestamp is found to be non-zero, this means that the resource is intended and just about to be deleted
	// hence, it's time to clean up the finalizers
	if !appObject.GetDeletionTimestamp().IsZero() {
		logger.Info("Deletion detected! Proceeding to cleanup the finalizers...")
		if err := r.deleteApplicationProfileFinalizerCallback(ctx, logger, appObject); err != nil {
			logger.Error(err, "error occurred while dealing with the delete finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	name, description, tenant_name := appObject.Spec.Name, appObject.Spec.Description, appObject.Spec.Tenant
	logger.Info(fmt.Sprintf("Creating Application Profile %s (%s)", name, description))

	if err = r.ApicClient.CreateApplicationProfile(name, description, tenant_name); err != nil {
		logger.Error(err, "error occurred while Creating Application Profile")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apicv1alpha1.ApplicationProfile{}).
		Complete(r)
}

func (r *ApplicationProfileReconciler) deleteApplicationProfileFinalizerCallback(ctx context.Context, logger logr.Logger, appObject *v1alpha1.ApplicationProfile) error {

	// delete the row with the above 'id' from the above 'table'
	if err := r.ApicClient.DeleteApplicationProfile(appObject.Spec.Name, appObject.Spec.Tenant); err != nil {
		return fmt.Errorf("error occurred while deleting Tenant: %w", err)
	}

	// remove the cleanup-row finalizer from the postgresWriterObject
	controllerutil.RemoveFinalizer(appObject, "finalizers.applicationprofiles.apic.aci.cisco/delete")
	if err := r.Update(ctx, appObject); err != nil {
		return fmt.Errorf("error occurred while removing the finalizer: %w", err)
	}
	logger.Info("cleaned up the 'finalizers.applicationprofiles.apic.aci.cisco/delete' finalizer successfully")
	return nil
}
