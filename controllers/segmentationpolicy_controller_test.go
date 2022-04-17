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
	"time"

	"github.com/jgomezve/aci-operator/api/v1alpha1"
	apicv1alpha1 "github.com/jgomezve/aci-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
	Context("When creating a new Segmentation Policy", func() {
		segPol := &apicv1alpha1.SegmentationPolicy{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apic.aci.cisco/v1alpha1",
				Kind:       "SegmentationPolicy",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      SegmentationPolicyName,
				Namespace: SegmentationPolicyNamespace,
			},
			Spec: apicv1alpha1.SegmentationPolicySpec{
				Name:       SegmentationPolicyName,
				Tenant:     SegmentationPolicyTenant,
				Namespaces: []string{"nsA", "nsB"},
				Rules: []v1alpha1.RuleSpec{
					{
						Eth:  "ip",
						IP:   "tcp",
						Port: 80,
					},
				},
			},
		}
		It("Should create APIC Filters when new Segmentation Policy is created", func() {
			By("Creating new Segmentation Policy", func() {
				ctx := context.Background()
				Expect(k8sClient.Create(ctx, segPol)).Should(Succeed())
				segPolLookupKey := types.NamespacedName{Name: SegmentationPolicyName, Namespace: SegmentationPolicyNamespace}
				createdSegPol := &apicv1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeTrue())
				Expect(createdSegPol.Spec.Name).Should(Equal(SegmentationPolicyName))
			})
		})
		It("Should delete APIC filters when a Segmentation Policy is deleted", func() {
			By("Deleting the newly Segmentation Policy", func() {
				Expect(k8sClient.Delete(ctx, segPol)).Should(Succeed())
				segPolLookupKey := types.NamespacedName{Name: SegmentationPolicyName, Namespace: SegmentationPolicyNamespace}
				createdSegPol := &apicv1alpha1.SegmentationPolicy{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, segPolLookupKey, createdSegPol)
					return err == nil
				}, timeout, interval).Should(BeFalse())
			})
		})
	})

})
