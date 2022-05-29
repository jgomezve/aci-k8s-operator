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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/jgomezve/aci-k8s-operator/api/v1alpha1"
	"github.com/jgomezve/aci-k8s-operator/pkg/aci"
	"github.com/jgomezve/aci-k8s-operator/pkg/utils"
)

var (
	finalizersSegPol = "finalizers.segmentationpolicies.apic.aci.cisco/delete"
)

// SegmentationPolicyReconciler reconciles a SegmentationPolicy object
type SegmentationPolicyReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	ApicClient aci.ApicInterface
	CniConfig  AciCniConfig
}

type AciCniConfig struct {
	ApicIp                        string
	ApicUsername                  string
	ApicPrivateKey                string
	KeyPath                       string
	PolicyTenant                  string
	PodBridgeDomain               string
	KubernetesVmmDomain           string
	EPGKubeDefault                string
	ApplicationProfileKubeDefault string
}

const (
	ApplicationProfileNamePrefix = "Seg_Pol_%s"
)

//+kubebuilder:rbac:groups=apic.aci.cisco,resources=segmentationpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=segmentationpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apic.aci.cisco,resources=segmentationpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups="",resources=pods/exec,verbs=create;

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
	err := r.Get(ctx, req.NamespacedName, segPolObject)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("SegmentationPolicy resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error occurred while fetching the Segmentation Policy resource")
		return ctrl.Result{}, err
	}

	segPolObject.Status.State = "Creating"
	err = r.Status().Update(context.Background(), segPolObject)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error occurred while setting the status: %w", err)
	}

	// if the event is not related to delete, just check if the finalizers are rightfully set on the resource
	if segPolObject.GetDeletionTimestamp().IsZero() && !controllerutil.ContainsFinalizer(segPolObject, finalizersSegPol) {
		// set the finalizers of the SegmentationPolicy to the rightful ones
		controllerutil.AddFinalizer(segPolObject, finalizersSegPol)
		if err := r.Update(ctx, segPolObject); err != nil {
			logger.Error(err, "error occurred while setting the finalizers of the SegmentationPolicy resource")
			return ctrl.Result{}, err
		}
	}

	// if the metadata.deletionTimestamp is found to be non-zero, this means that the resource is intended and just about to be deleted
	// hence, it's time to clean up the finalizers
	if !segPolObject.GetDeletionTimestamp().IsZero() && controllerutil.ContainsFinalizer(segPolObject, finalizersSegPol) {
		logger.Info("Deletion detected! Proceeding to cleanup the finalizers...")
		if err := r.deleteSegPolicyFinalizerCallback(ctx, logger, segPolObject); err != nil {
			logger.Error(err, "error occurred while dealing with the delete finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Reconcile K8s SegmentationPolicies' Namespaces and APIC EPGs
	result, err := r.ReconcileNamespacesEpgs(ctx, logger, segPolObject)
	if err != nil {
		return result, err
	}

	segPolObject.Status.State = "EPGs Created"
	err = r.Status().Update(context.Background(), segPolObject)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error occurred while setting the status: %w", err)
	}

	// Create Contract and Subject and associate the filters
	filtersSegPol := []string{}
	for _, rule := range segPolObject.Spec.Rules {
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
		filtersSegPol = append(filtersSegPol, filterName)
	}

	// Create contract (and subject) with all the filters listed in the SegmentationPolicy
	r.ApicClient.CreateContract(r.CniConfig.PolicyTenant, segPolObject.Name, filtersSegPol)
	logger.Info(fmt.Sprintf("Creating Contract/Subject %s", segPolObject.Name))

	// Read from the APIC the filters configured on the contract
	apicFilters, _ := r.ApicClient.GetContractFilters(segPolObject.Name, r.CniConfig.PolicyTenant)
	logger.Info(fmt.Sprintf("Contract Filters %s", apicFilters))

	// Delete/Update SubjectToFilter associations configured on the APIC but not listed in the SegmentationPolicy
	for _, apicFlt := range utils.Unique(filtersSegPol, apicFilters) {
		r.ApicClient.DeleteFilterFromSubjectContract(segPolObject.Name, r.CniConfig.PolicyTenant, apicFlt)
	}

	// Reconcile K8s SegmentationPolicies' Rules and APIC Filters
	result, err = r.ReconcileRulesFilters(logger, segPolObject)
	if err != nil {
		return result, err
	}
	segPolObject.Status.State = "Enforced"
	err = r.Status().Update(context.Background(), segPolObject)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error occurred while setting the status: %w", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SegmentationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SegmentationPolicy{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}},
			handler.EnqueueRequestsFromMapFunc(r.nameSpaceSegPolicyMapFunc)).
		//TODO: Makre the code convergent. Status attributed should only be modified if the APIC is actually modified
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// Generate SegmentationPolicy request based on changes in the K8s Namespaces
func (r *SegmentationPolicyReconciler) nameSpaceSegPolicyMapFunc(object client.Object) []reconcile.Request {
	modifiedNs := object.(*corev1.Namespace)
	logger := log.FromContext(context.TODO())
	logger.Info(fmt.Sprintf("Namespace %s modified", modifiedNs.Name))
	currentSegmentationPolicies := &v1alpha1.SegmentationPolicyList{}
	err := r.List(context.TODO(), currentSegmentationPolicies)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := []reconcile.Request{}
	for _, pol := range currentSegmentationPolicies.Items {
		for _, ns := range pol.Spec.Namespaces {
			if ns == modifiedNs.Name {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      pol.GetName(),
						Namespace: pol.GetNamespace(),
					},
				})
				logger.Info(fmt.Sprintf("Creating Reconcile request for SegmentationPolicy %s", pol.Name))
			}
		}
	}
	return requests
}

// Remove the APIC objects associated with a SegmentationPolicy
func (r *SegmentationPolicyReconciler) deleteSegPolicyFinalizerCallback(ctx context.Context, logger logr.Logger, segPolObject *v1alpha1.SegmentationPolicy) error {

	// TODO:  Update() call fails if the segPolObject is "used" (atributes are used to delete obejcts from the APIC) beforehand. Ask in Slack
	// remove finalizer
	controllerutil.RemoveFinalizer(segPolObject, finalizersSegPol)
	if err := r.Update(ctx, segPolObject); err != nil {
		return fmt.Errorf("error occurred while removing the finalizer: %w", err)
	}

	// Delete all the filters defined in the SegmenationPolicy
	for _, rule := range segPolObject.Spec.Rules {
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
		// Delete the Filter objects
		if err := r.ApicClient.DeleteFilter(r.CniConfig.PolicyTenant, filterName); err != nil {
			return fmt.Errorf("error occurred while deleting filter: %w", err)
		}
	}
	// Delete the contract and subject
	if err := r.ApicClient.DeleteContract(r.CniConfig.PolicyTenant, segPolObject.Name); err != nil {
		return fmt.Errorf("error occurred while deleting contract: %w", err)
	}

	// Check the EPGs associated with the SegmentationPolicy
	for _, nsPol := range segPolObject.Spec.Namespaces {
		logger.Info(fmt.Sprintf("EPG must be updated %s", nsPol))
		// Read the Annotation created on the EPG to check with SegmentationPolicies 'mananage' the EPG
		annotations, _ := r.ApicClient.GetAnnotationsEpg(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant)
		logger.Info(fmt.Sprintf("Annotations configured on EPG %s : %s", nsPol, annotations))
		// If the EPG only has one annotation (and the annotation that corresponds to the SegmenationPolicy), then delete the EPG
		if len(annotations) == 1 && annotations[0] == segPolObject.Name {
			logger.Info(fmt.Sprintf("Deleting EPG  %s", nsPol))
			r.ApicClient.DeleteEndpointGroup(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant)
			r.RemoveAnnotationNamesapce(ctx, nsPol)
			// If the EPG has more annotations, then remove the annotation that corresponds to the SegmentationPolicy, and stop consuming/providind the SegmentationPolicy's contract
		} else if len(annotations) > 1 {
			logger.Info(fmt.Sprintf("Removing annotation %s from EPG %s", segPolObject.Name, nsPol))
			r.ApicClient.RemoveTagAnnotation(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
			r.ApicClient.DeleteContractConsumer(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
			r.ApicClient.DeleteContractProvider(nsPol, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
		}
	}
	logger.Info(fmt.Sprintf("cleaned up the '%s' finalizer successfully", finalizersSegPol))
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

	// Set the status
	segPolObject.Status.Namespaces = strings.Join(utils.Intersect(nsClusterNames, segPolObject.Spec.Namespaces), ", ")
	err := r.Status().Update(context.Background(), segPolObject)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error occurred while setting the status: %w", err)
	}

	// Always create/overwrite the same Application Profile
	logger.Info(fmt.Sprintf("Creating Application Profile %s", segPolObject.Name))
	r.ApicClient.CreateApplicationProfile(fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), "", r.CniConfig.PolicyTenant)
	// Create EPGs for those namespaces listed in the SegmentationPolicy and configured on K8s
	for _, ns := range utils.Intersect(nsClusterNames, segPolObject.Spec.Namespaces) {
		if exists, _ := r.ApicClient.EpgExists(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant); exists {
			// If the EPG already exist, just add a new annotation. (An EPG/NS can be included in multiple policies)
			logger.Info(fmt.Sprintf("Adding annotation to EPG  %s", ns))
			r.ApicClient.AddTagAnnotationToEpg(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name, segPolObject.Name)
			// TODO: Unit Test error if Contracts are consumed/provided after the 'if' statement
			// Always consume/provide contracts
			logger.Info(fmt.Sprintf("Consume & Provide Segmentation Policy contract for EPG %s", ns))
			r.ApicClient.ConsumeContract(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
			r.ApicClient.ProvideContract(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
		} else {
			// If not, create the EPG and add annotation
			logger.Info(fmt.Sprintf("Creating EPG for Namespace %s", ns))
			r.ApicClient.CreateEndpointGroup(ns, "", fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, r.CniConfig.PodBridgeDomain, r.CniConfig.KubernetesVmmDomain)
			// TODO: Unit Test error if Contracts are consumed/provided after the 'if' statement
			// Always consume/provide contracts
			logger.Info(fmt.Sprintf("Consume & Provide Segmentation Policy contract for EPG %s", ns))
			r.ApicClient.ConsumeContract(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
			r.ApicClient.ProvideContract(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
			logger.Info(fmt.Sprintf("Adding annotation to EPG  %s", ns))
			r.ApicClient.AddTagAnnotationToEpg(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name, segPolObject.Name)
			logger.Info(fmt.Sprintf("Inheriting Contracts from ap-%s/epg-%s", r.CniConfig.ApplicationProfileKubeDefault, r.CniConfig.EPGKubeDefault))
			r.ApicClient.InheritContractFromMaster(ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, r.CniConfig.ApplicationProfileKubeDefault, r.CniConfig.EPGKubeDefault)
			logger.Info(fmt.Sprintf("Annotation K8s Namespace"))
			err := r.AnnotateNamespace(ctx, ns, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant)
			if err != nil {
				logger.Info(fmt.Sprintf("Error k8s annotation %s", err))
			}
		}
	}

	// Get EPGs configured on the APIC with the SegmentPolicy annotation
	epgApic, _ := r.ApicClient.GetEpgWithAnnotation(fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
	logger.Info(fmt.Sprintf("List of EPGs under Policy %s :  %s", segPolObject.Name, epgApic))
	// Delete/Update those EPGs configured on the APIC but not listed in the SegmentationPolicy
	for _, epg := range utils.Unique(utils.Intersect(nsClusterNames, segPolObject.Spec.Namespaces), epgApic) {
		logger.Info(fmt.Sprintf("EPG must be updated %s", epg))
		annotations, _ := r.ApicClient.GetAnnotationsEpg(epg, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant)
		logger.Info(fmt.Sprintf("Annotations configured on EPG %s : %s", epg, annotations))
		// If the EPG only has one annotation (and the annotation that corresponds to the SegmenationPolicy), then delete the EPG
		if len(annotations) == 1 && annotations[0] == segPolObject.Name {
			logger.Info(fmt.Sprintf("Deleting EPG  %s", epg))
			r.ApicClient.DeleteEndpointGroup(epg, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant)
			r.RemoveAnnotationNamesapce(ctx, epg)
			// If the EPG has more annotations, then remove the annotation that corresponds to the SegmentationPolicy, and stop consuming/providind the SegmentationPolicy's contract
		} else if len(annotations) > 1 {
			logger.Info(fmt.Sprintf("Removing annotation %s from EPG %s", segPolObject.Name, epg))
			r.ApicClient.RemoveTagAnnotation(epg, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
			r.ApicClient.DeleteContractConsumer(epg, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
			r.ApicClient.DeleteContractProvider(epg, fmt.Sprintf(ApplicationProfileNamePrefix, r.CniConfig.PolicyTenant), r.CniConfig.PolicyTenant, segPolObject.Name)
		}
	}
	return ctrl.Result{}, nil
}

// Reconcile the filters on the APIC based on the rules defined in the SegmentationPolicy
func (r *SegmentationPolicyReconciler) ReconcileRulesFilters(logger logr.Logger, segPolObject *v1alpha1.SegmentationPolicy) (ctrl.Result, error) {
	//Create Filters and filter entries based on the policy rules
	filtersSegPol := []string{}

	// Set the status
	segPolObject.Status.Rules = flattenRules(segPolObject.Spec.Rules)
	err := r.Status().Update(context.Background(), segPolObject)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Create Filters for those rules listed in the SegmentationPolicy
	for _, rule := range segPolObject.Spec.Rules {
		filterName := fmt.Sprintf("%s_%s%s%s", segPolObject.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
		logger.Info(fmt.Sprintf("Checking filter %s ", filterName))
		filtersSegPol = append(filtersSegPol, filterName)
		// Only create a filter if it does not exist already
		if exists, _ := r.ApicClient.FilterExists(filterName, r.CniConfig.PolicyTenant); !exists {
			logger.Info(fmt.Sprintf("Creating Filter %s", filterName))
			r.ApicClient.CreateFilterAndFilterEntry(r.CniConfig.PolicyTenant, filterName, rule.Eth, rule.IP, rule.Port)
			// Annotation is required to keep track of the filters SegmentationPolicy Object created on the APIC
			logger.Info(fmt.Sprintf("Tag Filter %s with annotation %s", filterName, segPolObject.Name))
			r.ApicClient.AddTagAnnotationToFilter(filterName, r.CniConfig.PolicyTenant, segPolObject.Name, segPolObject.Name)
		}
	}
	//Delete filters
	filtersApic, _ := r.ApicClient.GetFilterWithAnnotation(r.CniConfig.PolicyTenant, segPolObject.Name)
	logger.Info(fmt.Sprintf("List of filters under Policy %s :  %s", segPolObject.Name, filtersApic))
	for _, fltApic := range utils.Unique(filtersSegPol, filtersApic) {
		logger.Info(fmt.Sprintf("Deleting Filter %s", fltApic))
		r.ApicClient.DeleteFilter(r.CniConfig.PolicyTenant, fltApic)
	}
	return ctrl.Result{}, nil
}

func (r *SegmentationPolicyReconciler) AnnotateNamespace(ctx context.Context, nsName, appName, tenantName string) error {

	dnJson := fmt.Sprintf(`{\"tenant\":\"%s\",\"app-profile\":\"%s\",\"name\":\"%s\"}`, tenantName, appName, nsName)
	patch := []byte(fmt.Sprintf(`{"metadata":{"annotations":{"opflex.cisco.com/endpoint-group": "%s"}}}`, dnJson))
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	if err := r.Client.Patch(ctx, ns, client.RawPatch(types.MergePatchType, patch)); err != nil {
		return err
	}
	return nil
}

func (r *SegmentationPolicyReconciler) RemoveAnnotationNamesapce(ctx context.Context, nsName string) error {

	patch := []byte(`{"metadata":{"annotations":{"opflex.cisco.com/endpoint-group": ""}}}`)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	if err := r.Client.Patch(ctx, ns, client.RawPatch(types.MergePatchType, patch)); err != nil {
		return err
	}
	return nil
}
