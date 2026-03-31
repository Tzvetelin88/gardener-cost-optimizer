# Gardener Summary

This file summarizes what Gardener actually is, what it already supports, what is true in `PROJECT.md`, and where a new SAP-side extension can add value without duplicating existing Gardener functionality.

## Scope Of This Verification

The local workspace currently contains only `PROJECT.md`, not a cloned Gardener source tree. This summary is therefore verified against the upstream Gardener repository and official Gardener documentation:

- `https://github.com/gardener/gardener`
- `https://gardener.cloud/docs/gardener/concepts/architecture/`
- `https://gardener.cloud/docs/gardener/extensions/`
- `https://gardener.cloud/docs/gardener/concepts/gardenlet/`
- `https://gardener.cloud/docs/gardener/concepts/scheduler/`
- `https://gardener.cloud/docs/extensions/`

## What Gardener Is

Gardener is a Kubernetes-native platform for managing Kubernetes clusters as a service. It does not just provision raw VMs. Instead, it introduces its own APIs and controllers to create and operate Kubernetes clusters in a consistent way across multiple infrastructures.

Its core model is:

- `Garden` cluster: central management cluster with Gardener APIs and controllers
- `Seed` cluster: hosts many shoot control planes
- `Shoot` cluster: the end-user Kubernetes cluster

The major architectural idea is hosted control planes:

- the shoot control plane runs as pods inside a seed cluster
- the shoot itself mainly contains worker nodes
- this avoids dedicated master VMs for each cluster
- this improves upgradeability and day-2 operations

## What Gardener Already Supports

### Core Platform Capabilities

Gardener already covers:

- declarative Kubernetes cluster creation and reconciliation
- cluster updates and Kubernetes version upgrades
- worker pool management
- cluster hibernation for cost saving
- backup and restore workflows
- seed scheduling and placement logic
- seed-to-shoot control plane hosting
- out-of-tree extension contracts for provider-specific functionality

Important core components that are worth understanding:

- `gardener-apiserver`
- `gardener-controller-manager`
- `gardener-scheduler`
- `gardenlet`
- `gardener-operator`
- extension controllers

### Infrastructure And Cloud Support

Gardener already supports a strong provider-extension model. The known extension ecosystem includes infrastructure providers such as:

- AWS
- Azure
- GCP
- OpenStack
- vSphere
- Alicloud
- Equinix Metal
- MetalStack
- KubeVirt
- Hetzner Cloud

This means Gardener is already designed to create and operate clusters across multiple infrastructures, not just one cloud.

### Extension Ecosystem Support

Gardener already has extension categories and implementations for:

- infrastructure providers
- DNS providers
- operating systems
- network plugins
- container runtimes
- generic shoot services

Examples already covered in the ecosystem:

- networking: `Calico`, `Cilium`
- OS: `Garden Linux`, `Ubuntu`, `Flatcar`
- runtime: `gVisor`, `Kata`
- services: shoot DNS, certificates, OIDC, Falco, Flux, registry cache

## What Gardener Does And Does Not Cover

### Already Covered

If your idea is any of the following, Gardener already covers a large part of it:

- create a new Kubernetes cluster on AWS or Azure
- manage worker VMs indirectly through cluster worker pools
- prepare cloud networking needed for the cluster
- integrate storage and load balancer behavior through the Kubernetes/cloud provider stack
- operate the control plane lifecycle
- support multi-cloud cluster operations through provider extensions

### Not The Main Gardener Abstraction

These are not Gardener's primary product abstraction:

- generic standalone VM lifecycle as a raw IaaS API
- generic VPC/network provisioning product for arbitrary workloads
- direct high-level application PaaS for `Deployment`, `StatefulSet`, `DaemonSet`

Important nuance:

- Gardener creates and manages the Kubernetes cluster
- workloads such as `Deployment`, `StatefulSet`, `DaemonSet`, `Service`, PVCs, and network policies run inside the created shoot cluster
- so workloads are supported by Kubernetes on top of Gardener, but Gardener itself is not primarily a workload application platform API

## Verification Of `PROJECT.md`

### Verified As True Or Mostly True

The following claims in `PROJECT.md` are directionally correct and aligned with official Gardener docs:

- Gardener is Kubernetes-native and controller-driven
- it uses hosted control planes in seed clusters
- it supports multi-cloud operation via extensions
- it is designed around out-of-tree extensions
- it already covers cluster lifecycle concerns such as upgrades and hibernation
- it is a strong base for building a higher-level platform operator on top

### True But Needs More Precision

Some parts are good but slightly too broad:

- workload identity support exists in the ecosystem and provider integrations, but it should not be described as one perfectly uniform built-in feature across all clouds without checking provider specifics
- OIDC and OpenTelemetry should not be described as simple core-native categories in the same sense as infrastructure or worker extensions
- "create VM" should be translated to "manage worker machines and pools for a shoot cluster", because that is the actual Gardener abstraction

### Missing Important Findings

`PROJECT.md` should also account for:

- `gardenlet` as the seed-side agent
- `gardener-scheduler` assigning shoots to seeds
- the Garden/Seed/Shoot model explicitly
- dashboard and multi-project operating model
- extension webhooks that mutate provider-specific control plane manifests
- the fact that networking between seed and shoot includes VPN-based control plane communication constraints

## Best Opportunities To Build On Top

The best add-on is not another cluster provisioner. The best add-on is a higher-level enterprise platform API that uses Gardener as the cluster lifecycle engine.

### Strong Extension Ideas

1. `Environment` API

- input: cloud, region, Kubernetes version, worker pools, exposure mode, baseline policies
- output: a Gardener `Shoot` plus standard defaults

2. `PlatformProfile` or `ClusterBaseline`

- enforce security, observability, ingress, storage, cost, and network defaults
- apply standard policies to every newly created shoot

3. `WorkloadBundle` API

- provide one higher-level API that renders to `Deployment`, `StatefulSet`, `DaemonSet`, `Service`, ingress, PVCs, and related policies
- this creates real platform value above raw cluster creation

4. Cost Optimization Operator

- hibernation schedules
- TTL clusters
- preview environment expiry
- idle environment detection

5. Identity Broker

- expose a simple cross-cloud access policy API
- translate it to AWS/Azure/GCP identity mechanisms per provider

6. Tenant Landing Zone Automation

- namespace setup
- quotas
- RBAC
- network policies
- DNS and certificate defaults

### What To Avoid Building First

Avoid starting with:

- a new raw VM/network provisioner
- your own etcd/control plane lifecycle engine
- a parallel cluster API that duplicates Gardener's existing control model

## Best Product Direction For SAP

The clearest story is:

"Gardener gives us multi-cloud Kubernetes cluster lifecycle. Our SAP platform layer adds tenant-ready APIs, policies, workload abstractions, cost controls, and enterprise automation on top."

That is a stronger direction than rebuilding infrastructure features Gardener already has.

## Recommended MVP

A practical first version would be:

- `Environment` CRD -> creates a `Shoot`
- `PlatformProfile` CRD -> applies defaults and guardrails
- `WorkloadBundle` CRD -> deploys workloads into the target shoot
- optional `ExternalAccessPolicy` CRD -> cloud identity and external service access

This gives you:

- multi-cloud cluster creation
- higher-level workload platform APIs
- opinionated enterprise defaults
- a clean extension story without fighting Gardener

## Bottom Line

Gardener already covers the hard cluster-management problems:

- multi-cloud cluster lifecycle
- hosted control planes
- worker infrastructure orchestration
- extension contracts
- upgrades, backup/restore, and hibernation

The most valuable thing to add is a higher-level operator or extension layer focused on tenant experience, workload onboarding, policy automation, and enterprise day-2 operations.
