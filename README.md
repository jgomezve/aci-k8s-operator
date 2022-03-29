# ACI Kubernetes Operator

```yaml
apiVersion: apic.aci.cisco/v1alpha1
kind: Tenant
metadata:
  name: tenant-k8s
spec:
  name: tenant-k8s
  description: Tenant created by K8s
```

```
$ kubectl get crd
NAME                     CREATED AT
tenants.apic.aci.cisco   2022-03-29T11:11:25Z
$ kubectl get tenants
NAME         AGE
tenant-k8s   8m55s
```

Build using [Kubebuilder](https://book.kubebuilder.io/introduction.html)
