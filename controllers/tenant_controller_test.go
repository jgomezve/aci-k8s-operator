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

	apicv1alpha1 "github.com/jgomezve/aci-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Tenant controller", func() {
	const (
		TenantName        = "mytenant"
		TenantNamespace   = "default"
		TenantDescription = "mydescription"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)
	Context("A Context", func() {
		It("A It", func() {
			By("A By", func() {
				ctx := context.Background()
				tenant := &apicv1alpha1.Tenant{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "apic.aci.cisco/v1alpha1",
						Kind:       "Tenant",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      TenantName,
						Namespace: TenantNamespace,
					},
					Spec: apicv1alpha1.TenantSpec{
						Name:        TenantName,
						Description: TenantDescription,
					},
				}
				Expect(k8sClient.Create(ctx, tenant)).Should(Succeed())
				tenantLookupKey := types.NamespacedName{Name: TenantName, Namespace: TenantNamespace}
				createdTenant := &apicv1alpha1.Tenant{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, tenantLookupKey, createdTenant)
					if err != nil {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())

				Expect(createdTenant.Spec.Description).Should(Equal(TenantDescription))
				Expect(createdTenant.Spec.Name).Should(Equal(TenantName))
			})
		})
	})

})
