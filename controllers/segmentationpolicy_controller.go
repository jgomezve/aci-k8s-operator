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
	"github.com/jgomezve/aci-operator/pkg/aci"
	"github.com/jgomezve/aci-operator/pkg/utils"
)

var (
	finalizersSegPol []string = []string{"finalizers.segmentationpolicies.apic.aci.cisco/delete"}
)

// SegmentationPolicyReconciler reconciles a SegmentationPolicy object
type SegmentationPolicyReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	ApicClient aci.ApicInterface
}

const (
	ApplicationProfileNamePrefix = "Seg_Pol_%s"
)

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

	// TODO: Check Result
	// Reconcile K8s SegmentationPolicies' Namespaces and APIC EPGs
	_, err = r.ReconcileNamespacesEpgs(ctx, logger, segPolObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create Contract and Subject and associate the filters
	filters := []string{}
	for _, rule := range segPolObject.Spec.Rules {
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
		filters = append(filters, filterName)
	}
	r.ApicClient.CreateContract(segPolObject.Spec.Tenant, segPolObject.Name, filters)
	logger.Info(fmt.Sprintf("Creating Contract/Subject %s", segPolObject.Name))

	apicFilters, _ := r.ApicClient.GetContractFilters(segPolObject.Name, segPolObject.Spec.Tenant)
	logger.Info(fmt.Sprintf("Contract Filters %s", apicFilters))

	for _, apicFlt := range apicFilters {
		found := false
		for _, specFlt := range filters {
			if specFlt == apicFlt {
				found = true
			}
		}
		if !found {
			r.ApicClient.DeleteFilterFromSubjectContract(segPolObject.Name, segPolObject.Spec.Tenant, apicFlt)
		}
	}

	// Reconcile K8s SegmentationPolicies' Rules and APIC Filters
	_, err = r.ReconcileRulesFilters(logger, segPolObject)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SegmentationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SegmentationPolicy{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.nameSpaceSegPolicyMapFunc)).
		Complete(r)
}

func (r *SegmentationPolicyReconciler) nameSpaceSegPolicyMapFunc(object client.Object) []reconcile.Request {
	modifiedNs := object.(*corev1.Namespace)
	// logger := log.FromContext(context.TODO())
	fmt.Printf("Namespace  %s modified", modifiedNs.Name)

	currentSegmentationPolicies := &v1alpha1.SegmentationPolicyList{}
	err := r.List(context.TODO(), currentSegmentationPolicies)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := []reconcile.Request{}
	for _, pol := range currentSegmentationPolicies.Items {
		for _, ns := range pol.Spec.Namespaces {
			if ns == modifiedNs.Name {
				fmt.Printf("Create Request for SegmentationPolicy %s", pol.Name)
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      pol.GetName(),
						Namespace: pol.GetNamespace(),
					},
				})
			}
		}
	}
	fmt.Printf("Requests: %s", requests)
	return requests
}

func (r *SegmentationPolicyReconciler) deleteSegPolicyFinalizerCallback(ctx context.Context, logger logr.Logger, segPolObject *v1alpha1.SegmentationPolicy) error {

	for _, rule := range segPolObject.Spec.Rules {
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
		// Delete the Filter objects
		if err := r.ApicClient.DeleteFilter(segPolObject.Spec.Tenant, filterName); err != nil {
			return fmt.Errorf("error occurred while deleting filter: %w", err)
		}
	}
	// Delete the contract and subject
	if err := r.ApicClient.DeleteContract(segPolObject.Spec.Tenant, segPolObject.Name); err != nil {
		return fmt.Errorf("error occurred while deleting contract: %w", err)
	}

	// Delete Annotation or EPGs
	for _, nsPol := range segPolObject.Spec.Namespaces {
		logger.Info(fmt.Sprintf("EPG must be updated %s", nsPol))
		annotations, err := r.ApicClient.GetAnnotationsEpg(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant)
		logger.Info(fmt.Sprintf("Annotations configured on EPG %s : %s", nsPol, annotations))
		if err != nil {
			return err
		}
		if len(annotations) == 1 && annotations[0] == segPolObject.Name {
			logger.Info(fmt.Sprintf("Deleting EPG  %s", nsPol))
			if err := r.ApicClient.DeleteEndpointGroup(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant); err != nil {
				return err
			}
		} else if len(annotations) > 1 {
			logger.Info(fmt.Sprintf("Removing annotation %s from EPG %s", segPolObject.Name, nsPol))
			if err := r.ApicClient.RemoveTagAnnotation(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name); err != nil {
				return err
			}
			r.ApicClient.DeleteContractConsumer(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
			r.ApicClient.DeleteContractProvider(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
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

// Reconcile the EPGs on the APIC based on the SegmentationPolicy definition
func (r *SegmentationPolicyReconciler) ReconcileNamespacesEpgs(ctx context.Context, logger logr.Logger, segPolObject *v1alpha1.SegmentationPolicy) (ctrl.Result, error) {

	// Read the Namespaces configured on K8s
	nsClusterConf := &corev1.NamespaceList{}
	r.List(ctx, nsClusterConf)
	nsClusterNames := []string{}
	for _, ns := range nsClusterConf.Items {
		nsClusterNames = append(nsClusterNames, ns.Name)
	}

	// Always create/overwrite the same Application Profile
	logger.Info(fmt.Sprintf("Creating Application Profile %s", segPolObject.Name))
	r.ApicClient.CreateApplicationProfile(fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), "", segPolObject.Spec.Tenant)
	// Create EPGs for those namespaces listed in the SegmentationPolicy and configured on K8s
	for _, ns := range utils.Intersect(nsClusterNames, segPolObject.Spec.Namespaces) {
		if exists, _ := r.ApicClient.EpgExists(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant); exists {
			// If the EPG already exist, just add a new annotation. (An EPG/NS can be included in multiple policies)
			logger.Info(fmt.Sprintf("Adding annotation to EPG  %s", ns))
			r.ApicClient.AddTagAnnotationToEpg(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name, segPolObject.Name)
		} else {
			// If not, create the EPG and add annotation
			logger.Info(fmt.Sprintf("Creating EPG for Namespace %s", ns))
			r.ApicClient.CreateEndpointGroup(ns, "", fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant)
			logger.Info(fmt.Sprintf("Adding annotation to EPG  %s", ns))
			r.ApicClient.AddTagAnnotationToEpg(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name, segPolObject.Name)
		}
		// Always consume/provide contracts
		r.ApicClient.ConsumeContract(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
		r.ApicClient.ProvideContract(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
	}

	// Get EPGs configured on the APIC with the SegmentPolicy annotation
	epgApic, _ := r.ApicClient.GetEpgWithAnnotation(fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
	logger.Info(fmt.Sprintf("List of EPGs under Policy %s :  %s", segPolObject.Name, epgApic))
	// Delete/Update those EPGs configued on the APIC but not listed in the SegmentationPolicy
	for _, epg := range utils.Unique(utils.Intersect(nsClusterNames, segPolObject.Spec.Namespaces), epgApic) {
		logger.Info(fmt.Sprintf("EPG must be updated %s", epg))
		annotations, _ := r.ApicClient.GetAnnotationsEpg(epg, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant)
		logger.Info(fmt.Sprintf("Annotations configured on EPG %s : %s", epg, annotations))
		// If the EPG only has one annotation (and the annotation that corresponds to the SegmenationPolicy), the delete the EPG
		if len(annotations) == 1 && annotations[0] == segPolObject.Name {
			logger.Info(fmt.Sprintf("Deleting EPG  %s", epg))
			r.ApicClient.DeleteEndpointGroup(epg, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant)
			// If the EPG has more annotations, then remove the annotation that corresponds to the SegmentationPolicy, and stop consuming/providind the SegmentationPolicy's contract
		} else if len(annotations) > 1 {
			logger.Info(fmt.Sprintf("Removing annotation %s from EPG %s", segPolObject.Name, epg))
			r.ApicClient.RemoveTagAnnotation(epg, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
			r.ApicClient.DeleteContractConsumer(epg, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
			r.ApicClient.DeleteContractProvider(epg, fmt.Sprintf(ApplicationProfileNamePrefix, segPolObject.Spec.Tenant), segPolObject.Spec.Tenant, segPolObject.Name)
		}
	}
	return ctrl.Result{}, nil
}

func (r *SegmentationPolicyReconciler) ReconcileRulesFilters(logger logr.Logger, segPolObject *v1alpha1.SegmentationPolicy) (ctrl.Result, error) {
	//Create Filters and filter entries based on the policy rules
	filtersC := []string{}
	for _, rule := range segPolObject.Spec.Rules {
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
		logger.Info(fmt.Sprintf("Checking filter %s ", filterName))
		filtersC = append(filtersC, filterName)
		exists, err := r.ApicClient.FilterExists(filterName, segPolObject.Spec.Tenant)
		logger.Info(fmt.Sprintf("Result FilterExists %s:%s ", strconv.FormatBool(exists), err))
		if err != nil {
			return ctrl.Result{}, err
		}
		// Only create a filter if it does not exist already
		if !exists {
			logger.Info(fmt.Sprintf("Creating Filter %s", filterName))
			r.ApicClient.CreateFilterAndFilterEntry(segPolObject.Spec.Tenant, filterName, rule.Eth, rule.IP, rule.Port)
			logger.Info(fmt.Sprintf("Creating Filter %s", filterName))
			r.ApicClient.AddTagAnnotationToFilter(filterName, segPolObject.Spec.Tenant, segPolObject.Name, segPolObject.Name)
		} else {
			logger.Info(fmt.Sprintf("Filter %s already exists ", filterName))
		}
	}
	//Delete filters
	filters, err := r.ApicClient.GetFilterWithAnnotation(segPolObject.Spec.Tenant, segPolObject.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info(fmt.Sprintf("List of Filters under Policy %s :  %s", segPolObject.Name, filters))
	for _, fltApic := range filters {
		toDel := true
		for _, fltK8s := range filtersC {
			if fltApic == fltK8s {
				toDel = false
			}
		}
		if toDel {
			logger.Info(fmt.Sprintf("Deleting Filter %s", fltApic))
			r.ApicClient.DeleteFilter(segPolObject.Spec.Tenant, fltApic)
		}
	}
	return ctrl.Result{}, nil
}
