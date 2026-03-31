# Smart Cost Optimizer

Smart Cost Optimizer is a platform project built on top of Gardener and Kubernetes.

The idea is to use Gardener as the cluster lifecycle engine and add a higher-level optimization layer that:

- discovers underused clusters and workloads
- estimates savings opportunities across clusters
- recommends hibernation, consolidation, and cheaper placement actions
- exposes APIs that can be consumed by a dashboard
- allows approved actions to be triggered from the UI or API

Gardener already manages Kubernetes clusters as a service with hosted control planes and a `Shoot`-based API, which makes it a strong foundation for this kind of optimizer ([Gardener repository](https://github.com/gardener/gardener)).

## UI Preview

Initial recommendations view:

![Smart Cost Optimizer UI](./app.png)

After executing a recommendation, the dashboard updates the savings summary, reduces ready actions, and records the completed change in action history:

![Smart Cost Optimizer After Action](./app_action.png)

## Goal

The project focuses on cost-aware operations across Gardener-managed Kubernetes clusters.

In scope for this MVP:

- cluster inventory across clouds and regions
- cost and utilization summaries
- optimization recommendations
- manual actions for safe operations
- a React dashboard for operators
- support for both mock data and real Gardener integration

The long-term direction is:

- make cluster costs visible
- recommend the cheapest relevant location for stateless workloads
- identify clusters that can be hibernated
- recommend consolidation candidates based on relevance, isolation, utilization, and savings
- support a gradual move from local demo mode to real Gardener environments

## What We Use

### Backend

- Go
- Kubernetes-style project layout
- Gardener `Shoot` integration
- REST API
- in-memory mock landscape for local development

Key backend areas:

- `backend/cmd/api`: API entrypoint
- `backend/cmd/operator`: background recommendation loop
- `backend/internal/gardener`: real and mock cluster data sources
- `backend/internal/metrics`: workload and utilization collection
- `backend/internal/recommender`: savings and recommendation logic
- `backend/internal/actions`: manual action execution
- `backend/openapi/openapi.yaml`: API contract

### Frontend

- React
- TypeScript
- Vite

Key frontend areas:

- `frontend/src/app`: top-level app shell
- `frontend/src/features/clusters`: cluster inventory
- `frontend/src/features/recommendations`: recommendations UI
- `frontend/src/features/actions`: action history
- `frontend/src/features/summary`: savings summary cards

### Delivery

- `docker-compose.yml` for local mock mode
- `DOCKER.md` for container-based setup and run instructions
- `scripts/dev.sh` for local development
- `scripts/dev.ps1` as an optional Windows fallback
- `Makefile` for common commands
- Helm chart under `deploy/helm/smart-cost-optimizer`

## Product Idea

The optimizer sits above Gardener and Kubernetes:

1. Read cluster inventory from Gardener.
2. Read or simulate workload/utilization signals.
3. Estimate cluster and workload costs.
4. Produce recommendations such as:
   - hibernate idle non-prod clusters
   - scale down oversized worker pools
   - move stateless workloads to cheaper clusters
   - recommend consolidation of similar low-utilization clusters
5. Expose the results through a backend API and React dashboard.
6. Allow operators to run approved actions.

This project is intentionally not a Gardener fork and not a custom provider extension first. It is a platform layer above Gardener.

## Data Source Modes

The backend supports three modes through the `DATA_SOURCE` environment variable.

### `mock`

Use a built-in in-memory landscape.

Use this when:

- you do not have Gardener yet
- you want to demo the product
- you want frontend and API development without cluster access

In `mock` mode:

- cluster inventory comes from built-in sample data
- workload metrics come from built-in sample workloads
- actions such as hibernation, scale-down, and workload moves are simulated in memory

### `real`

Use a real Gardener environment only.

Use this when:

- you have a working Gardener landscape
- you want real `Shoot` inventory
- you want real actions against Gardener or real shoot clusters

In `real` mode:

- the backend expects Gardener access through `GARDENER_KUBECONFIG`
- workload move actions require target/source shoot access through `SHOOT_KUBECONFIG_MAP`
- startup fails if Gardener is unavailable

### `auto`

Try real Gardener first, then fall back to mock mode.

Use this when:

- you want one build that works both locally and in integrated environments
- you want easy migration from demo mode to real mode

In `auto` mode:

- the backend attempts to connect to Gardener
- if connection fails and fallback is allowed, it switches to mock mode

## Configuration

Main backend environment variables:

- `API_ADDR`: backend listen address, default `:8080`
- `DATA_SOURCE`: `mock`, `real`, or `auto`
- `ENABLE_FALLBACK_DATA`: legacy fallback switch, still supported
- `GARDENER_KUBECONFIG`: path to kubeconfig for the Garden cluster
- `GARDENER_CONTEXT`: optional kubeconfig context
- `SHOOT_KUBECONFIG_MAP`: comma-separated mapping for shoot access
- `PROMETHEUS_URL`: optional metrics endpoint
- `FRONTEND_ORIGIN`: CORS origin for the dashboard
- `REFRESH_INTERVAL_SECONDS`: recommendation refresh interval

Example `SHOOT_KUBECONFIG_MAP`:

```text
dev-aws-a=C:\kubeconfigs\dev-aws-a.yaml,dev-aws-b=C:\kubeconfigs\dev-aws-b.yaml
```

## API Surface

Current endpoints:

- `GET /api/v1/clusters`
- `GET /api/v1/recommendations`
- `GET /api/v1/recommendations/:id`
- `GET /api/v1/actions`
- `GET /api/v1/savings/summary`
- `POST /api/v1/actions/hibernate-cluster`
- `POST /api/v1/actions/scale-nodepool`
- `POST /api/v1/actions/move-workload`

The full API contract lives in `backend/openapi/openapi.yaml`.

### Example API Output

Health check:

```text
http://localhost:8080/healthz -> ok
```

Recommendations example from mock mode:

```json
[
  {
    "id": "move-dev-aws-a-web-catalog-api",
    "kind": "cheaper-placement",
    "subject": "web/catalog-api",
    "reason": "Stateless workload is running on a more expensive cluster than a comparable lower-cost target.",
    "evidence": [
      "Workload is stateless",
      "Target cluster is cheaper",
      "Target cluster utilization remains below 70%"
    ],
    "monthlySavings": 116.4,
    "risk": "medium",
    "executable": true,
    "sourceCluster": "garden-dev/dev-aws-a",
    "targetCluster": "garden-dev/dev-aws-b",
    "targetWorkload": "web/catalog-api",
    "actionType": "move-workload",
    "createdAt": "2026-03-31T14:24:02.919629716Z"
  },
  {
    "id": "hibernate-dev-aws-b",
    "kind": "idle-cluster",
    "subject": "dev-aws-b",
    "reason": "Cluster has no discovered workloads and has stayed mostly idle.",
    "evidence": [
      "Idle score above 75",
      "No active deployments discovered",
      "Suitable for non-prod hibernation"
    ],
    "monthlySavings": 78.9568,
    "risk": "low",
    "executable": true,
    "targetCluster": "garden-dev/dev-aws-b",
    "actionType": "hibernate-cluster",
    "createdAt": "2026-03-31T14:24:02.919629716Z"
  }
]
```

## Local Development

### Option 1: Docker Compose

This is the easiest way to run the demo locally.

```bash
docker compose up --build
```

Current defaults in `docker-compose.yml` start the API in `mock` mode.

After startup:

- frontend: `http://localhost:5173`
- backend: `http://localhost:8080`

### Option 2: Shell Script

Preferred local runner:

```bash
DATA_SOURCE=mock sh ./scripts/dev.sh
```

You can also choose:

```bash
DATA_SOURCE=auto sh ./scripts/dev.sh
DATA_SOURCE=real sh ./scripts/dev.sh
```

After startup:

- frontend: `http://localhost:4173`
- backend: `http://localhost:8080`

Notes:

- the script installs frontend dependencies automatically
- the script builds the frontend and starts a local preview server on `http://localhost:4173`
- the backend only starts if `go` is installed and available on `PATH`

### Option 3: PowerShell Fallback

If you prefer PowerShell on Windows:

```powershell
.\scripts\dev.ps1 -DataSource mock
```

You can also choose:

```powershell
.\scripts\dev.ps1 -DataSource auto
.\scripts\dev.ps1 -DataSource real
```

### Option 4: Makefile

Common commands:

```bash
make frontend-install
make frontend-build
make backend-run
make frontend-run
make run
make run-ps
make docker-up
make helm-template
```

## Running Against Real Gardener

When you are ready to move from mock mode to a real Gardener environment:

1. Set `DATA_SOURCE=real` or `DATA_SOURCE=auto`.
2. Provide `GARDENER_KUBECONFIG`.
3. Optionally provide `GARDENER_CONTEXT`.
4. Provide `SHOOT_KUBECONFIG_MAP` for workload-level actions.
5. Start the backend.

Example on Windows PowerShell:

```powershell
$env:DATA_SOURCE="real"
$env:GARDENER_KUBECONFIG="C:\kubeconfigs\garden.yaml"
$env:GARDENER_CONTEXT="garden-admin"
$env:SHOOT_KUBECONFIG_MAP="dev-aws-a=C:\kubeconfigs\dev-aws-a.yaml,dev-aws-b=C:\kubeconfigs\dev-aws-b.yaml"
cd .\backend
go run ./cmd/api
```

## Helm Deployment

The project includes a Helm chart:

- chart path: `deploy/helm/smart-cost-optimizer`

Default Helm values use:

- `DATA_SOURCE: "real"`
- `ENABLE_FALLBACK_DATA: "false"`

Render the manifests locally with:

```bash
make helm-template
```

If you want to deploy in mock mode for demos, override the values:

```bash
helm template smart-cost-optimizer ./deploy/helm/smart-cost-optimizer --set backend.env.DATA_SOURCE=mock --set backend.env.ENABLE_FALLBACK_DATA=true
```

## Current Limitations

- real backend compilation and execution require Go to be installed locally
- real workload move actions require per-shoot Kubernetes access
- pricing is currently heuristic and not yet backed by full cloud billing APIs
- metrics are basic and can later be enriched with Prometheus or cloud cost sources

## Suggested Workflow

If you are just starting:

1. Run in `mock` mode.
2. Validate the dashboard and API behavior.
3. Refine recommendation logic and UI.
4. Switch to `auto` mode during early integration.
5. Move to `real` mode once Gardener and shoot kubeconfigs are ready.

This gives you a safe path from demo to real Gardener operations without changing the overall architecture.
