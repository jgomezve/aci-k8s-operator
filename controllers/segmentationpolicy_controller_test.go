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
	ctx := context.Background()
	Namespaces := []string{"ns-a", "ns-b"}

	Context("When creating a new Segmentation Policy", func() {
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
		It("Should create APIC Objects when new Segmentation Policy is created", func() {
			By("Creating a new K8s Namespace", func() {
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
		It("Should delete APIC Objects when a Segmentation Policy is deleted", func() {
			By("Deleting the newly Segmentation Policy", func() {
				Expect(k8sClient.Delete(ctx, segPol)).Should(Succeed())
				segPolLookupKey := types.NamespacedName{Name: SegmentationPolicyName, Namespace: SegmentationPolicyNamespace}
				createdSegPol := &v1alpha1.SegmentationPolicy{}
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
			})
			By("Checking deleted APIC EPGs", func() {
				// TODO: Not implemeted Yet
				// TODO: Keep the name logic out of the Test. Test using mock-only functions
				// for _, ns := range Namespaces {
				// 	Eventually(func() bool {
				// 		exists, _ := apicClient.EpgExists(ns, fmt.Sprintf("Seg_Pol_%s", segPol.Spec.Tenant), segPol.Spec.Tenant)
				// 		return exists
				// 	}, timeout, interval).Should(BeFalse())
				// }
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
