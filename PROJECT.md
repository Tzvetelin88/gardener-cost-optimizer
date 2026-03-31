# Project Architecture

## Context

Gardener is a Kubernetes cluster lifecycle manager. It creates and manages shoot clusters across cloud providers, handles upgrades, hibernation, and worker pool scaling. What Gardener explicitly does not include is a cost intelligence layer: there is no cross-cluster utilization analysis, no savings estimation, and no optimization recommendations.

This project fills that gap by building an intelligence and action layer on top of Gardener.

The mental model is:

```
Gardener          →  cluster lifecycle + execution platform
Smart Cost Optimizer  →  intelligence + recommendations + operator actions
```

## What the Product Does

The optimizer does four things:

1. Reads cluster inventory from Gardener (or from a built-in mock landscape).
2. Evaluates cluster utilization, workload placement, and estimated cost.
3. Produces recommendations with evidence and monthly savings estimates.
4. Lets operators execute safe cost-saving actions through a REST API and React dashboard.

Recommendation kinds:

- `idle-cluster` — cluster has no significant workloads and is a candidate for hibernation
- `cheaper-placement` — a stateless workload is running on a more expensive cluster than necessary
- `cluster-consolidation` — two clusters with similar purpose and low utilization can be merged
- `scale-nodepool` — a worker pool is oversized relative to current demand

Executable actions (in both mock and real mode):

- hibernate a cluster
- wake a hibernated cluster
- move a stateless workload to a cheaper cluster
- scale a worker pool min/max

## Architecture

The backend is a single Go service with clear internal boundaries:

```
cmd/api          → HTTP server entrypoint
internal/config  → environment-based configuration
internal/gardener → data source: real Gardener client and mock fallback
internal/metrics  → workload and utilization signals per cluster
internal/pricing  → heuristic cost catalog per cloud/region/machine type
internal/recommender → recommendation engine: scoring, evidence, savings
internal/actions  → action service: execute and persist operator decisions
internal/http    → REST API handlers and middleware
internal/models  → shared domain types
api/v1alpha1     → CRD-style type definitions (Recommendation, Action, OptimizationPolicy)
```

The frontend is a React single-page application served by Nginx in the Docker setup, or by Vite in development mode. It calls the backend REST API and renders four main areas: cluster inventory, recommendations, action history, and savings summary.

## Data Source Layer

One of the key design decisions is that the backend does not require Gardener to be running. The `DATA_SOURCE` variable selects how cluster and workload data is loaded:

- `mock` — a built-in landscape with sample clusters and workloads, all mutations in-memory
- `real` — connects to a real Garden cluster using a kubeconfig; reads live `Shoot` objects
- `auto` — tries real first, silently falls back to mock

This makes the product runnable on any laptop without any external dependencies, and it gives a clean upgrade path to a real Gardener environment without changing the product architecture.

## Action Persistence

Completed actions are written to a JSONL file (`data/actions.jsonl` by default). On restart, the service reloads this history. This means the action log survives container restarts without needing a database.

## Tech Stack

**Backend:**

- Go with a Kubernetes-style project layout
- `internal/` for all business logic (not exposed as a library)
- `api/v1alpha1/` for typed domain objects following Kubernetes CRD conventions
- Gin for HTTP routing
- Standard Kubernetes client-go libraries for Gardener integration

**Frontend:**

- React + TypeScript
- Vite for development and build
- Nginx for serving the built app in Docker
- Flat feature-folder layout: `features/clusters`, `features/recommendations`, `features/actions`, `features/summary`

**Delivery:**

- Docker Compose for local cross-platform demo (no local Go or Node required)
- Shell script (`scripts/dev.sh`) and PowerShell script (`scripts/dev.ps1`) for local development
- Makefile for common commands
- Helm chart under `deploy/helm/smart-cost-optimizer` for Kubernetes deployment

## Design Decisions

**No database required.** Cluster state in mock mode is held in memory. Action history is a flat JSONL file. This keeps local setup to a single `docker compose up`.

**Recommendations are always re-computed.** The recommendation engine reruns on a configurable interval. There is no persistent recommendation store; the engine is the source of truth.

**Consolidation is advisory, not executable.** Merging two clusters involves team, data, and dependency decisions that cannot be automated safely. The product surfaces the opportunity and estimated savings, but leaves the decision to the operator.

**Heuristic pricing.** Cluster costs are estimated from a pricing catalog based on machine type, cloud, and region. This is intentionally approximate. The value is in relative comparisons (cluster A is cheaper than B), not absolute billing numbers.

**Thresholds are configurable.** `IDLE_THRESHOLD` and `TARGET_UTILIZATION` can be tuned via environment variables, so the recommendation sensitivity can be adjusted without changing code.

## Scope and Limits

Current scope:

- cluster inventory from Gardener or mock
- recommendation engine for four decision types
- executable actions for hibernation, wake, move, and scale
- persistent action history
- React dashboard
- REST API with full OpenAPI spec

Known limits:

- pricing is heuristic, not backed by cloud billing APIs
- metrics are derived from Kubernetes resource requests, not actual Prometheus usage
- consolidation is recommendation-only
- stateful workload migration is not supported
- `OptimizationPolicy` CRD type is defined but no controller processes it yet

## Positioning

This project is not a Gardener fork. It does not modify Gardener internals. It sits above Gardener and calls the same APIs that operators already use.

The long-term direction is to make cluster costs visible, recommend the cheapest relevant placement for stateless workloads, and gradually extend action coverage as confidence in the recommendation engine grows.
