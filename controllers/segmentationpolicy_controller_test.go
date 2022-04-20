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

var _ = Describe("Segmentation Policy controller", func() {
	const (
		SegmentationPolicyName      = "segpol1"
		SegmentationPolicyNamespace = "default"
		SegmentationPolicyTenant    = "k8s-tenant"
		timeout                     = time.Second * 10
		duration                    = time.Second * 10
		interval                    = time.Millisecond * 250
	)

	Namespaces := []string{"ns-a", "ns-b"}
	ctx := context.Background()
	segPol := &v1alpha1.SegmentationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apic.aci.cisco/v1alpha1",
			Kind:       "SegmentationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      SegmentationPolicyName,
			Namespace: SegmentationPolicyNamespace,
		},
		Spec: v1alpha1.SegmentationPolicySpec{
			Tenant:     SegmentationPolicyTenant,
			Namespaces: Namespaces,
			Rules: []v1alpha1.RuleSpec{
				{
					Eth:  "ip",
					IP:   "tcp",
					Port: 80,
				},
			},
		},
	}
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
			Namespaces: []string{"ns-b", "ns-c", "ns-d"},
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

	Context("When creating the first Segmentation Policy", func() {

		It("Should create APIC Objects when new Segmentation Policy is created", func() {
			By("Creating new K8s Namespaces", func() {
				for _, ns := range Namespaces {
					ns_a := &corev1.Namespace{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Namespace",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: ns,
						},
					}
					Expect(k8sClient.Create(ctx, ns_a)).Should(Succeed())
				}
			})
			By("Creating a new K8s Segmentation Policy", func() {
				Expect(k8sClient.Create(ctx, segPol)).Should(Succeed())
				segPolLookupKey := types.NamespacedName{Name: SegmentationPolicyName, Namespace: SegmentationPolicyNamespace}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				Expect(createdSegPol.Name).Should(Equal(SegmentationPolicyName))
			})
			By("Checking created APIC Filters", func() {
				for _, rule := range segPol.Spec.Rules {
					// TODO: Keep the name logic out of the Test. Test using mock-only functions
					filterName := fmt.Sprintf("%s_%s%s%s", segPol.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
			})
			By("Checking created APIC EPGs", func() {
				// TODO: Keep the name logic out of the Test. Test using mock-only functions
				for _, ns := range Namespaces {
					Eventually(func() bool {
						exists, _ := apicClient.EpgExists(ns, fmt.Sprintf("Seg_Pol_%s", segPol.Spec.Tenant), segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
			})
		})
	})

	Context("Creating an additional Segmentation Policy", func() {
		It("Should create additional APIC Objects when an additional Segmentation Policy is created", func() {
			By("Creating additional K8s Namespacse", func() {
				for _, ns := range []string{"ns-c", "ns-d", "ns-f"} {
					ns := &corev1.Namespace{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "v1",
							Kind:       "Namespace",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: ns,
						},
					}
					Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
				}
			})
			By("Creating the additional Segmentation Policies", func() {

				Expect(k8sClient.Create(ctx, segPol2)).Should(Succeed())
				segPolLookupKey := types.NamespacedName{Name: "segpol2", Namespace: "default"}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				Expect(createdSegPol.Name).Should(Equal("segpol2"))
			})
			By("Checking created APIC EPGs for both Segmentation Policies", func() {
				// TODO: Keep the name logic out of the Test. Test using mock-only functions
				for _, ns := range segPol.Spec.Namespaces {
					Eventually(func() bool {
						exists, _ := apicClient.EpgExists(ns, fmt.Sprintf("Seg_Pol_%s", segPol.Spec.Tenant), segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
				for _, ns := range segPol2.Spec.Namespaces {
					Eventually(func() bool {
						exists, _ := apicClient.EpgExists(ns, fmt.Sprintf("Seg_Pol_%s", segPol2.Spec.Tenant), segPol2.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
			})
			By("Checking created APIC Filters for both Segmentation Policies", func() {
				for _, rule := range segPol.Spec.Rules {
					// TODO: Keep the name logic out of the Test. Test using mock-only functions
					filterName := fmt.Sprintf("%s_%s%s%s", segPol.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
				for _, rule := range segPol2.Spec.Rules {
					// TODO: Keep the name logic out of the Test. Test using mock-only functions
					filterName := fmt.Sprintf("%s_%s%s%s", segPol2.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())

				}
			})
			By("Checking EPG with multiple tags", func() {
				tags, _ := apicClient.GetAnnotationsEpg("ns-b", fmt.Sprintf("Seg_Pol_%s", segPol.Spec.Tenant), segPol2.Spec.Tenant)
				sort.Strings(tags)
				Expect(tags).Should(Equal([]string{"segpol1", "segpol2"}))

			})
		})
	})

	Context("Updating an existing Segmentation Policy", func() {
		It("Should update EPGs and Filters on the APIC", func() {
			By("Updating an existing Segmentation Policy", func() {

				segPolLookupKey := types.NamespacedName{Name: "segpol2", Namespace: "default"}
				queriedObj := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, queriedObj)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				// First Query the Object to ge the current Resource Version
				segPol2_1.ObjectMeta.ResourceVersion = queriedObj.ObjectMeta.ResourceVersion
				// Update the Object with the new configuration
				Expect(k8sClient.Update(ctx, segPol2_1)).Should(Succeed())
				// Check the Object still exists
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				Expect(createdSegPol.Name).Should(Equal("segpol2"))
			})

			By("Checking a EPG has been deleted", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-d", fmt.Sprintf("Seg_Pol_%s", segPol2.Spec.Tenant), segPol2.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeFalse())
			})
			By("Checking a Tag has been removed from an EPG", func() {
				tags, _ := apicClient.GetAnnotationsEpg("ns-b", fmt.Sprintf("Seg_Pol_%s", segPol.Spec.Tenant), segPol2.Spec.Tenant)
				Expect(tags).Should(Equal([]string{"segpol1"}))
			})
			By("Checking a Filter has been Deleted", func() {
				filterName := fmt.Sprintf("%s_%s%s%s", segPol.Name, "ip", "icmp", "0")
				Eventually(func() bool {
					exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeFalse())
			})
			By("Checking all the APIC Filters exits", func() {
				for _, rule := range segPol2_1.Spec.Rules {
					// TODO: Keep the name logic out of the Test. Test using mock-only functions
					filterName := fmt.Sprintf("%s_%s%s%s", segPol2_1.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeTrue())
				}
			})
			By("Checking a new EPG exist", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-f", fmt.Sprintf("Seg_Pol_%s", segPol2.Spec.Tenant), segPol2.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeTrue())
			})
			By("Checking a EPG was created", func() {
				Eventually(func() bool {
					exists, _ := apicClient.EpgExists("ns-e", fmt.Sprintf("Seg_Pol_%s", segPol2.Spec.Tenant), segPol2.Spec.Tenant)
					return exists
				}, timeout, interval).Should(BeFalse())
			})
		})
	})

	Context("Delete all existing Segmentation Policies", func() {
		It("Should delete APIC Objects when a Segmentation Policy is deleted", func() {
			By("Deleting all existing Segmentation Policies", func() {
				Expect(k8sClient.Delete(ctx, segPol)).Should(Succeed())
				segPolLookupKey := types.NamespacedName{Name: "segpol1", Namespace: "default"}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeFalse())

				Expect(k8sClient.Delete(ctx, segPol2_1)).Should(Succeed())
				segPolLookupKey = types.NamespacedName{Name: "segpol2", Namespace: "default"}
				createdSegPol = &v1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeFalse())
			})
			By("Checking deleted APIC filters", func() {
				for _, rule := range segPol.Spec.Rules {
					// TODO: Keep the name logic out of the Test. Test using mock-only functions
					filterName := fmt.Sprintf("%s_%s%s%s", segPol.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeFalse())
				}
				for _, rule := range segPol2_1.Spec.Rules {
					// TODO: Keep the name logic out of the Test. Test using mock-only functions
					filterName := fmt.Sprintf("%s_%s%s%s", segPol2_1.Name, rule.Eth, rule.IP, strconv.Itoa(rule.Port))
					Eventually(func() bool {
						exists, _ := apicClient.FilterExists(filterName, segPol2_1.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeFalse())
				}
			})
			By("Checking deleted APIC EPGs", func() {
				//TODO: Keep the name logic out of the Test. Test using mock-only functions
				for _, ns := range segPol.Spec.Namespaces {
					Eventually(func() bool {
						exists, _ := apicClient.EpgExists(ns, fmt.Sprintf("Seg_Pol_%s", segPol.Spec.Tenant), segPol.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeFalse())
				}
				for _, ns := range segPol2_1.Spec.Namespaces {
					Eventually(func() bool {
						exists, _ := apicClient.EpgExists(ns, fmt.Sprintf("Seg_Pol_%s", segPol2_1.Spec.Tenant), segPol2_1.Spec.Tenant)
						return exists
					}, timeout, interval).Should(BeFalse())
				}
			})
			By("Checking K8s Namespaces are left untouched", func() {
				namespaces := &corev1.NamespaceList{}
				k8sClient.List(ctx, namespaces)
				var found bool
				for _, nsManifest := range Namespaces {
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
