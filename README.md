# ACI Kubernetes Operator

[![Tests](https://github.com/jgomezve/aci-k8s-operator/actions/workflows/test.yml/badge.svg)](https://github.com/jgomezve/aci-k8s-operator/actions/workflows/test.yml)

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
