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

* Clone this repository

      git clone https://github.com/jgomezve/aci-k8s-operator
      cd aci-k8s-operator

* Configure the `SegmentationPolicy` Custom Resource Definition (CRD)

      make install

```
$ kubectl get crd
NAME                                    CREATED AT
segmentationpolicies.apic.aci.cisco     2022-04-19T15:58:11Z
```

> **_NOTE:_** The CRD manifest is located used `config/crd/bases/apic.aci.cisco_segmentationpolicies.yaml`

* Start the operator
 
### Option 1: Operator running outside of the K8s Cluster

* Set the APIC credentials as environmental variables

      export APIC_HOST=<apic_ip_address>
      export APIC_USERNAME=<apic_username>
      export APIC_PASSWORD=<apic_password>

* Compile the code and execute the binary file 

      make run

```
1.6509721382001133e+09	INFO	setup	starting manager
1.6509721382004604e+09	INFO	controller.segmentationpolicy	Starting EventSource	{"reconciler group": "apic.aci.cisco", "reconciler kind": "SegmentationPolicy", "source": "kind source: *v1alpha1.SegmentationPolicy"}
1.650972138200524e+09	INFO	controller.segmentationpolicy	Starting EventSource	{"reconciler group": "apic.aci.cisco", "reconciler kind": "SegmentationPolicy", "source": "kind source: *v1.Namespace"}
1.6509721382005382e+09	INFO	controller.segmentationpolicy	Starting Controller	{"reconciler group": "apic.aci.cisco", "reconciler kind": "SegmentationPolicy"}
1.6509721383018e+09	INFO	controller.segmentationpolicy	Starting workers	{"reconciler group": "apic.aci.cisco", "reconciler kind": "SegmentationPolicy", "worker count": 1}
```
### Option 2: Operator running as a Container in the K8s Cluster
      
      make deploy

### Usage 


* Create four namespaces

      kubectl create ns <ns_name>

```
NAME                    STATUS   AGE
aci-containers-system   Active   145d
cattle-system           Active   145d
default                 Active   145d
fleet-system            Active   145d
ingress-nginx           Active   145d
kube-node-lease         Active   145d
kube-public             Active   145d
kube-system             Active   145d
ns1                     Active   11s
ns2                     Active   8s
ns3                     Active   5s
ns4                     Active   3s

```

* Configure a two `SegmentationPolicy` Custom Resources (CR) 

*apic_v1alpha1_segmentationpolicy.yaml*
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
---
apiVersion: apic.aci.cisco/v1alpha1
kind: SegmentationPolicy
metadata:
  name: segpol2
spec:
  tenant: k8s-operator
  namespaces:
    - ns2
    - ns3
    - ns4
  rules:
    - eth: ip
      ip: tcp
      port: 80 
```

      kubectl apply -f apic_v1alpha1_segmentationpolicy.yaml

```
$ kubectl get segmentationpolicies.apic.aci.cisco
NAME      AGE
segpol1   22s
segpol2   22s
```
> **_NOTE:_** Examples can be found under `config/samples/apic_v1alpha1_segmentationpolicy.yaml`

![add-app](docs/images/aci_topology.png "ACI Topology")

## More Information
Build using [Kubebuilder](https://book.kubebuilder.io/introduction.html)
