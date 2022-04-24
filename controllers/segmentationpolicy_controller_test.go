/*
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
// +kubebuilder:docs-gen:collapse=Apache License

package controllers

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jgomezve/aci-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Segmentation Policy DOES NOT manage/own K8s Namespaces.
// It creates the corresponding ACI Objects if the defined Namespace exists.
// It deletes the corresponding ACI Object if a linked Namespaces is removed

var _ = Describe("Segmentation Policy controller", func() {
	const (
		SegmentationPolicyNamespace = "default"
		SegmentationPolicyTenant    = "k8s-tenant"
		timeout                     = time.Second * 10
		duration                    = time.Second * 10
		interval                    = time.Millisecond * 250
	)

	ctx := context.Background()

	// Namespaces of SegmentationPolicy #1
	nsSegPol1 := []string{"ns-a", "ns-b", "ns-c"}
	// SegmentationPolicy #1
	segPol1 := &v1alpha1.SegmentationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apic.aci.cisco/v1alpha1",
			Kind:       "SegmentationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "segpol1",
			Namespace: SegmentationPolicyNamespace,
		},
		Spec: v1alpha1.SegmentationPolicySpec{
			Tenant:     SegmentationPolicyTenant,
			Namespaces: nsSegPol1,
			Rules: []v1alpha1.RuleSpec{
				{
					Eth:  "ip",
					IP:   "tcp",
					Port: 80,
				},
			},
		},
	}

	// Namespaces of SegmentationPolicy #2
	nsSegPol2 := []string{"ns-b", "ns-c", "ns-d"}
	// Namespaces created in  K8s before creating SegmentationPolicy #2
	nsK8sSegPol2 := []string{"ns-d", "ns-f"}
	// SegmentationPolicy #2
	segPol2 := &v1alpha1.SegmentationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apic.aci.cisco/v1alpha1",
			Kind:       "SegmentationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "segpol2",
			Namespace: "default",
		},
		Spec: v1alpha1.SegmentationPolicySpec{
			Tenant:     "k8s-tenant",
			Namespaces: nsSegPol2,
			Rules: []v1alpha1.RuleSpec{
				{
					Eth:  "ip",
					IP:   "tcp",
					Port: 80,
				},
				{
					Eth: "ip",
					IP:  "icmp",
				},
			},
		},
	}

	// SegmentationPolicy #2 (Updated)
	segPol2_1 := &v1alpha1.SegmentationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apic.aci.cisco/v1alpha1",
			Kind:       "SegmentationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "segpol2",
			Namespace: "default",
		},
		Spec: v1alpha1.SegmentationPolicySpec{
			Tenant:     "k8s-tenant",
			Namespaces: []string{"ns-c", "ns-e", "ns-f"},
			Rules: []v1alpha1.RuleSpec{
				{
					Eth:  "ip",
					IP:   "tcp",
					Port: 80,
				},
				{
					Eth: "arp",
				},
			},
		},
	}

	// For the SegmentationPolicy #1 all the defined Namespaces already exist in the K8s cluster
	Context("When creating the first Segmentation Policy", func() {

		It("Should create APIC Objects when new Segmentation Policy is created", func() {
			// Create in the K8s Cluster the Namespaces specified in the Segmentation Policy
			By("Creating new K8s Namespaces", func() {
				for _, ns := range nsSegPol1 {
					newNs := &corev1.Namespace{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Namespace",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: ns,
						},
					}
					Expect(k8sClient.Create(ctx, newNs)).Should(Succeed())
				}
			})
			By("Creating a new K8s Segmentation Policy", func() {
				// Create SegmentationPolicy #1
				Expect(k8sClient.Create(ctx, segPol1)).Should(Succeed())
				// Verify the SegmentationPolicy #1 is created in K8s
				segPolLookupKey := types.NamespacedName{Name: segPol1.Name, Namespace: SegmentationPolicyNamespace}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				Expect(createdSegPol.Name).Should(Equal(segPol1.Name))
			})
			By("Checking Contracts and filters in the APIC", func() {
				filters := []string{}
				for _, rule := range segPol1.Spec.Rules {
					filterName := fmt.Sprintf("%s_%s%s%s", segPol1.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					filters = append(filters, filterName)
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol1.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
				apicFilters, _ := apicClient.GetContractFilters(segPol1.Name, segPol1.Spec.Tenant)
				Expect(apicFilters).Should(Equal(filters))
			})
			By("Checking created APIC EPGs", func() {
				for _, ns := range segPol1.Spec.Namespaces {
					Eventually(func() bool {
						exists, _ := apicClient.EpgExists(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
			})
			By("Checking contracts consumed/provided by EPG", func() {
				for _, ns := range segPol1.Spec.Namespaces {
					contracts, _ := apicClient.GetContracts(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
					Expect(contracts["consumed"]).Should(Equal([]string{segPol1.Name}))
					Expect(contracts["provided"]).Should(Equal([]string{segPol1.Name}))
				}
			})
		})
	})

	// For the SegmentationPolicy #2 not all the defined Namespaces exist in the K8s Cluster.
	// There is also a Namespaces defined in SegmentationPolicy #1 and SegmentationPolicy #2
	Context("When creating an additional Segmentation Policy", func() {

		It("Should create additional APIC Objects when an additional Segmentation Policy is created", func() {
			// Create in the K8s Cluster additional Namespaces. The new Namespaces do not match the ones stated in the Segmentation Policy #2
			By("Creating additional K8s Namespacse", func() {
				for _, ns := range nsK8sSegPol2 {
					newNs := &corev1.Namespace{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Namespace",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: ns,
						},
					}
					Expect(k8sClient.Create(ctx, newNs)).Should(Succeed())
				}
			})
			By("Creating the additional Segmentation Policies", func() {
				// Create SegmentationPolicy #2
				Expect(k8sClient.Create(ctx, segPol2)).Should(Succeed())
				// Verify the SegmentationPolicy #1 is created in K8s
				segPolLookupKey := types.NamespacedName{Name: segPol2.Name, Namespace: SegmentationPolicyNamespace}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				Expect(createdSegPol.Name).Should(Equal(segPol2.Name))
			})
			By("Checking created APIC EPGs for both Segmentation Policies", func() {

				for _, segPol := range []v1alpha1.SegmentationPolicy{*segPol1, *segPol2} {
					for _, ns := range segPol.Spec.Namespaces {
						Eventually(func() bool {
							exists, _ := apicClient.EpgExists(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPol.Spec.Tenant), segPol.Spec.Tenant)
							return exists
						}, timeout, interval).Should(BeTrue())
					}
				}
			})
			By("Checking created APIC Filters for both Segmentation Policies", func() {
				for _, segPol := range []v1alpha1.SegmentationPolicy{*segPol1, *segPol2} {
					filters := []string{}
					for _, rule := range segPol.Spec.Rules {
						filterName := fmt.Sprintf("%s_%s%s%s", segPol.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
						filters = append(filters, filterName)
						Eventually(func() bool {
							exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
							return exists
						}, timeout, interval).Should(BeTrue())
					}
					apicFilters, _ := apicClient.GetContractFilters(segPol.Name, segPol.Spec.Tenant)
					fmt.Printf("Filters Contract %s : %s", segPol.Name, apicFilters)
					Expect(apicFilters).Should(Equal(filters))
				}
			})
			// Namespaces defined in both Segmentation Policies should have two Tag Annotation. One per Segmentation Policy
			By("Checking EPG with multiple tags", func() {
				tags, _ := apicClient.GetAnnotationsEpg("ns-b", fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
				sort.Strings(tags)
				Expect(tags).Should(Equal([]string{segPol1.Name, segPol2.Name}))

			})
			By("Checking EPG providing and consuming multiple contracts", func() {
				contracts, _ := apicClient.GetContracts("ns-b", fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
				Expect(contracts["consumed"]).Should(Equal([]string{segPol1.Name, segPol2.Name}))
				Expect(contracts["provided"]).Should(Equal([]string{segPol1.Name, segPol2.Name}))
			})
		})
	})

	// The updated version of SegmentationPolicy #2 no longer includes the Namespaces already defined in SegmentationPolicy #
	// The updated version of SegmentationPolicy #2 no longer defines a K8s Namespaces. The corresponding EPG is deleted
	// The updated version of SegmentationPolicy #2 defines another K8s Namespaces. A new EPG is created
	// The updated version of SegmentationPolicy #2 defines a Namespaces that does not exist in K8s. Nothing should happen
	Context("When updating an existing Segmentation Policy", func() {

		It("Should update EPGs and Filters on the APIC", func() {
			// Update SegmentationPolicy #2.
			By("Updating an existing Segmentation Policy", func() {
				segPolLookupKey := types.NamespacedName{Name: segPol2.Name, Namespace: "default"}
				queriedObj := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, queriedObj)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				// First query the SegmentationPolicy to get the current Resource Version
				segPol2_1.ObjectMeta.ResourceVersion = queriedObj.ObjectMeta.ResourceVersion
				// Update the SegmentationPolicy with the new configuration
				Expect(k8sClient.Update(ctx, segPol2_1)).Should(Succeed())
				// Check the SegmentationPolicy still exists
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				Expect(createdSegPol.Name).Should(Equal(segPol2.Name))
			})
			By("Checking a EPG has been deleted", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-d", fmt.Sprintf(ApplicationProfileNamePrefix, segPol2.Spec.Tenant), segPol2.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeFalse())
			})
			By("Checking a Tag has been removed from an EPG", func() {
				tags, _ := apicClient.GetAnnotationsEpg("ns-b", fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol2.Spec.Tenant)
				Expect(tags).Should(Equal([]string{segPol1.Name}))
			})
			By("Checking EPG no longer consumes a Contract", func() {
				contracts, _ := apicClient.GetContracts("ns-b", fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
				Expect(contracts["consumed"]).Should(Equal([]string{segPol1.Name}))
				Expect(contracts["provided"]).Should(Equal([]string{segPol1.Name}))
			})
			By("Checking a Filter has been Deleted", func() {
				filterName := fmt.Sprintf("%s_%s%s%s", segPol1.Name, "ip", "icmp", "0")
				Eventually(func() bool {
					exists, _ := apicClient.FilterExists(filterName, segPol1.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeFalse())
			})
			By("Checking all the APIC Filters exits", func() {
				filters := []string{}
				for _, rule := range segPol2_1.Spec.Rules {
					filterName := fmt.Sprintf("%s_%s%s%s", segPol2_1.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					filters = append(filters, filterName)
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol1.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
				apicFilters, _ := apicClient.GetContractFilters(segPol2_1.Name, segPol1.Spec.Tenant)
				fmt.Printf("Filters Contract %s : %s", segPol2_1.Name, apicFilters)
				Expect(apicFilters).Should(Equal(filters))
			})
			// TODO. Calculate dynamically the affected K8s Namespaces by comparing the list Namespaces in the Segmentation Policies
			By("Checking a new EPG exist", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-f", fmt.Sprintf(ApplicationProfileNamePrefix, segPol2.Spec.Tenant), segPol2.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeTrue())
			})
			By("Checking a EPG was not created", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-e", fmt.Sprintf(ApplicationProfileNamePrefix, segPol2.Spec.Tenant), segPol2.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeFalse())
			})
		})
	})

	// Delete SegmentationPolicy #1
	Context("When deleting a Segmentation Policy", func() {
		It("Should delete contracts and update EPGs", func() {
			By("Deleting  a Segmentation Policy", func() {
				Expect(k8sClient.Delete(ctx, segPol1)).Should(Succeed())
				segPolLookupKey := types.NamespacedName{Name: segPol1.Name, Namespace: "default"}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeFalse())
			})
			By("Checking deleted APIC filters", func() {
				for _, rule := range segPol1.Spec.Rules {
					filterName := fmt.Sprintf("%s_%s%s%s", segPol1.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol1.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeFalse())
				}
			})
			By("Checking that EPGs only managed by that SegmentationPolicy are deleted", func() {
				for _, ns := range []string{"ns-a", "ns-b"} {
					Eventually(func() bool {
						exists, _ := apicClient.EpgExists(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeFalse())
				}
			})
			By("Checking that EPGs managed by other SegmentationPolicy are not deleted", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-c", fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeTrue())
			})
			By("Checking that EPG only consumes contract associated with SegmentationPolcy 1", func() {
				contracts, _ := apicClient.GetContracts("ns-c", fmt.Sprintf(ApplicationProfileNamePrefix, segPol1.Spec.Tenant), segPol1.Spec.Tenant)
				Expect(contracts["consumed"]).Should(Equal([]string{segPol2.Name}))
				Expect(contracts["provided"]).Should(Equal([]string{segPol2.Name}))
			})
		})
	})

	// Create K8s Namespaces alreay listed in a SegmentationPolicy
	Context("When creating a new namespace, which is defined in a SegmentationPolicy", func() {
		It("Should update the affected Segmentation Policies", func() {
			By("Creating new K8s Namespaces", func() {
				newNs := &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns-e",
					},
				}
				Expect(k8sClient.Create(ctx, newNs)).Should(Succeed())
			})
			By("Checking a EPG has been created", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-e", fmt.Sprintf(ApplicationProfileNamePrefix, segPol2.Spec.Tenant), segPol2.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeTrue())
			})
			By("Checking the EPG consumes/provides contract associated with the Segmenation Policy", func() {
				contracts, _ := apicClient.GetContracts("ns-e", fmt.Sprintf(ApplicationProfileNamePrefix, segPol2.Spec.Tenant), segPol2.Spec.Tenant)
				Expect(contracts["consumed"]).Should(Equal([]string{segPol2.Name}))
				Expect(contracts["provided"]).Should(Equal([]string{segPol2.Name}))
			})
			By("Checking EPG with the corresponding tags", func() {
				tags, _ := apicClient.GetAnnotationsEpg("ns-f", fmt.Sprintf(ApplicationProfileNamePrefix, segPol2.Spec.Tenant), segPol2.Spec.Tenant)
				sort.Strings(tags)
				Expect(tags).Should(Equal([]string{segPol2.Name}))

			})
		})
	})

	// Delete K8s Namespaces alreay listed in a SegmentationPolicy
	// TODO: Namespaces cannot be deleted from EnvTest. See https://github.com/kubernetes-sigs/controller-runtime/issues/880
	Context("When deleting a namespace, which is degined in multiple Segmentation Policies", func() {
		It("Should update the affected Segmentation Policies", func() {
			// By("Deleting new K8s Namespaces", func() {
			// 	newNs := &corev1.Namespace{
			// 		TypeMeta: metav1.TypeMeta{
			// 			APIVersion: "v1",
			// 			Kind:       "Namespace",
			// 		},
			// 		ObjectMeta: metav1.ObjectMeta{
			// 			Name: "ns-c",
			// 		},
			// 	}
			// 	Expect(k8sClient.Delete(ctx, newNs)).Should(Succeed())
			// })
			// By("Checking a EPG has been deleted", func() {
			// 	Eventually(func() bool {
			// 		exists, _ := apicClient.EpgExists("ns-c", fmt.Sprintf(ApplicationProfileNamePrefix, segPol2.Spec.Tenant), segPol2.Spec.Tenant)
			// 		return exists
			// 	}, timeout, interval).Should(BeFalse())
			// })
		})
	})

	// Delete SegmentationPolicy #2
	Context("When deleting the  remaining Segmentation Policy", func() {

		It("Should delete APIC Objects when a Segmentation Policy is deleted", func() {
			// DeleteSegmentationPolicy #2.
			By("Deleting remaining Segmentation Policies", func() {
				Expect(k8sClient.Delete(ctx, segPol2)).Should(Succeed())

				segPolLookupKey := types.NamespacedName{Name: segPol2.Name, Namespace: "default"}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeFalse())
			})
			By("Checking deleted APIC filters", func() {
				for _, segPol := range []v1alpha1.SegmentationPolicy{*segPol1, *segPol2} {
					for _, rule := range segPol.Spec.Rules {
						filterName := fmt.Sprintf("%s_%s%s%s", segPol.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
						Eventually(func() bool {
							exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
							return exists
						}, timeout, interval).Should(BeFalse())
					}
				}
			})
			By("Checking deleted APIC EPGs", func() {
				for _, segPol := range []v1alpha1.SegmentationPolicy{*segPol1, *segPol2} {
					for _, ns := range segPol.Spec.Namespaces {
						Eventually(func() bool {
							exists, _ := apicClient.EpgExists(ns, fmt.Sprintf(ApplicationProfileNamePrefix, segPol.Spec.Tenant), segPol.Spec.Tenant)
							return exists
						}, timeout, interval).Should(BeFalse())
					}
				}
			})
			By("Checking K8s Namespaces are left untouched", func() {
				namespaces := &corev1.NamespaceList{}
				k8sClient.List(ctx, namespaces)
				var found bool
				// Not all the Namespaces defined in th Segmentation Policy #2 are actually configured in K8s
				for _, nsManifest := range append(nsSegPol1, nsK8sSegPol2...) {
					found = false
					for _, nsK8s := range namespaces.Items {
						if nsManifest == nsK8s.Name {
							found = true
						}
					}
					Expect(found).Should(BeTrue())
				}
			})
		})
	})
})
