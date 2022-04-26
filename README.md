# ACI Kubernetes Operator
[![Tests](https://github.com/jgomezve/aci-k8s-operator/actions/workflows/test.yaml/badge.svg)](https://github.com/jgomezve/aci-k8s-operator/actions/workflows/test.yaml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jgomezve/aci-k8s-operator)
![Kubernetes version](https://img.shields.io/badge/kubernetes-1.23%2B-blue)

Define Network Segmentation Policies as Kubernetes Resources, enforces them on the ACI Fabric with the APIC Controller.

This repository contains a Kubernetes Operator used to manage Kubernetes Namespaces Segmentation Rules, while are later enforced on the ACI Fabric by means ACI Constructs (EPGs, Contracts, Filter)

## Requirements

* [Cisco APIC](https://www.cisco.com/c/en/us/solutions/data-center-virtualization/application-centric-infrastructure/index.html) >= 5.2.x 
* [Kubernetes](https://kubernetes.io/) >= 1.23
* [ACI-CNI](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/aci/apic/sw/kb/b_Kubernetes_Integration_with_ACI.html)
* [Go](https://golang.org/doc/install) >= 1.17 (Optional)



## Installation


### Option 1: Operator running outside of the K8s Cluster


### Option 2: Operator running as a Container in the K8s Cluster



```yaml
apiVersion: apic.aci.cisco/v1alpha1
kind: SegmentationPolicy
metadata:
  name: segpol1
spec:
  tenant: k8s-operator
  namespaces:
    - ns1
    - ns2
  rules:
    - eth: arp
    - eth: ip
      ip: udp
      port: 53
```
Build using [Kubebuilder](https://book.kubebuilder.io/introduction.html)
