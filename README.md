# ACI Kubernetes Operator
[![Tests](https://github.com/jgomezve/aci-k8s-operator/actions/workflows/test.yaml/badge.svg)](https://github.com/jgomezve/aci-k8s-operator/actions/workflows/test.yaml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jgomezve/aci-k8s-operator)
![Kubernetes version](https://img.shields.io/badge/kubernetes-1.23%2B-blue)

Define Network Segmentation Policies as Kubernetes Resources and enforce them on the ACI Fabric with the APIC Controller.

This repository contains a Kubernetes Operator used to manage Kubernetes Namespaces Segmentation Rules, while are later enforced on the ACI Fabric by means ACI Constructs (EPGs, Contracts, Filter)

## Overview

The Cisco ACI-CNI takes care of enablinng network connectivity between Pods by provisioning network resources on the worker nodes and on the ACI Fabric. The ACI-CNI extends the ACI iVXLAN Encapsulation used on the ACI Fabric down to the OpenVSwitches running on the worker nodes. This integration gives tha ACI Fabric more visibility into the Pod Network, therefore Pods are learned as endpoints of the ACI Fabric. It also allows you to 



Even though the ACI-CNI allows Kubernetes Administrators to map Namespaces/Deployments to ACI Endpoint Group, further Policy definition on the ACI Fabric stills requires... . This operators aims to automate the provisioning of ACI Constructs by definition a new Kubernetes Resource (Segmentation Policy)

 ## Requirements

* [Cisco APIC](https://www.cisco.com/c/en/us/solutions/data-center-virtualization/application-centric-infrastructure/index.html) >= 5.2.x 
* [Kubernetes](https://kubernetes.io/) >= 1.23
* [ACI-CNI](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/aci/apic/sw/kb/b_Kubernetes_Integration_with_ACI.html)
* [Go](https://golang.org/doc/install) >= 1.17 (Optional)



## Installation

**Your Kubernetes cluster must have already been configured to use the Cisco ACI CNI**

* Configure the `SegmentationPolicy` Custom Resource Definition (CRD)

*segmentation_policy_crd.yaml*
```yaml
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.8.0
  creationTimestamp: null
  name: segmentationpolicies.apic.aci.cisco
spec:
  group: apic.aci.cisco
  names:
    kind: SegmentationPolicy
    listKind: SegmentationPolicyList
    plural: segmentationpolicies
    singular: segmentationpolicy
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: SegmentationPolicy is the Schema for the segmentationpolicies
          API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: SegmentationPolicySpec defines the desired state of SegmentationPolicy
            properties:
              namespaces:
                items:
                  type: string
                type: array
              rules:
                items:
                  properties:
                    eth:
                      type: string
                    ip:
                      type: string
                    port:
                      type: integer
                  type: object
                type: array
              tenant:
                minLength: 0
                type: string
            required:
            - namespaces
            - rules
            - tenant
            type: object
          status:
            description: SegmentationPolicyStatus defines the observed state of SegmentationPolicy
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
```

      kubectl apply -f segmentation_policy_crd.yaml

> **_NOTE:_** Alternatively you could execute `make install`

* Start the operator
 
### Option 1: Operator running outside of the K8s Cluster


### Option 2: Operator running as a Container in the K8s Cluster

* Configure a `SegmentationPolicy` Custom Resource (CR) 

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
