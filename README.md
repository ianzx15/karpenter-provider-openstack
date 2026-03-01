# Karpenter Provider for OpenStack

This repository contains the implementation of a **Karpenter** provider for the **OpenStack** cloud platform. The goal of this project is to enable dynamic and efficient autoscaling for Kubernetes clusters running on OpenStack infrastructure, eliminating the dependency on static Node Groups and optimizing resource utilization through *just-in-time* provisioning.

This project was developed as a Final Course Project (TCC) at the **Federal University of Campina Grande (UFCG)**.

---

## 🚀 About the Project

Karpenter is a next-generation autoscaler that watches for "Pending" pods and provisions nodes directly in the cloud infrastructure. While established implementations exist for AWS, Azure, and GCP, this project fills the gap for the **OpenStack** ecosystem by utilizing the **Adapter Pattern** to integrate Kubernetes APIs with IaaS services.

[Image of Karpenter architecture workflow showing pending pods leading to node provision]

### Key Features
* **Node Group-less Provisioning:** Creates instances on-demand with the exact specifications required by the workload.
* **Intelligent Flavor Selection:** Maps CPU/RAM requirements to the most cost-effective *Flavor* in the OpenStack catalog.
* **Modular Architecture:** Written in Go, leveraging the **Gophercloud** SDK for interaction with Nova (Compute), Neutron (Networking), and Keystone (Identity) services.

## 🏗️ Architecture and Workflow

The provider acts as a bridge between the Karpenter controller and the OpenStack API. The workflow follows these steps:

1. **Detection:** Karpenter identifies a Pod that cannot be scheduled due to insufficient resources.
2. **Resolution:** The provider queries the `OpenStackNodeClass` and filters available `Flavors`.
3. **Provisioning:** The controller triggers `Nova` (OpenStack Compute) to instantiate the VM.
4. **Binding:** The instance is registered back to the Kubernetes cluster with the `ProviderID` format (`openstack:///VM-UUID`).

[Image of OpenStack logical architecture showing Nova, Neutron and Keystone services]

## 🛠️ Tech Stack

* **Language:** [Go (Golang)](https://go.dev/)
* **OpenStack SDK:** [Gophercloud](https://github.com/gophercloud/gophercloud)
* **Orchestration:** [Kubernetes](https://kubernetes.io/)

## 📖 Deploy

👉 **[Development & Deployment Guide](deploy/local/README.md)**
*Learn how to configure OpenStack credentials, install CRDs, and run the controller locally.*
