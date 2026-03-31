# Project Structure

## Top-Level Layout

```
SCO/
├── backend/                  Go backend service
├── frontend/                 React frontend
├── deploy/                   Helm chart for Kubernetes deployment
├── scripts/                  Local dev startup scripts
├── docker-compose.yml        Local Docker stack
├── Makefile                  Common dev commands
├── backend/openapi/          OpenAPI spec
└── *.md                      Documentation
```

## Backend

```
backend/
├── cmd/
│   ├── api/
│   │   └── main.go           HTTP server entrypoint; loads config, wires dependencies, starts Gin
│   └── operator/
│       └── main.go           Background recommendation loop (standalone runner)
│
├── api/
│   └── v1alpha1/
│       ├── action_types.go              Action domain type (ClusterAction, ActionRequest, ActionResult)
│       ├── recommendation_types.go      Recommendation type (kind, evidence, savings, risk)
│       └── optimizationpolicy_types.go  OptimizationPolicy CRD type (defined, controller pending)
│
├── internal/
│   ├── config/
│   │   └── config.go         Reads all environment variables into a typed Config struct
│   │
│   ├── gardener/
│   │   ├── client.go         Real Gardener client; connects to Garden cluster and lists Shoots
│   │   └── fallback.go       Mock landscape; built-in sample clusters and workloads
│   │
│   ├── metrics/
│   │   └── provider.go       Workload and utilization signals per cluster (requests-based, no Prometheus)
│   │
│   ├── pricing/
│   │   └── catalog.go        Heuristic cost catalog; estimates monthly cost by cloud/region/machine type
│   │
│   ├── recommender/
│   │   └── engine.go         Recommendation engine; scores clusters, produces typed recommendations with evidence and savings
│   │
│   ├── actions/
│   │   ├── service.go        Action service; executes hibernate, wake, move-workload, scale-nodepool
│   │   ├── service_test.go
│   │   └── transfer.go       Workload transfer logic (real mode: clone deployment, scale source)
│   │
│   ├── http/
│   │   ├── server.go         Gin router setup; all API handler functions
│   │   └── server_test.go
│   │
│   └── models/
│       └── models.go         Shared domain types: Cluster, Workload, Metrics, Summary
│
├── data/
│   └── actions.jsonl         Persistent action history (written on each action, reloaded on startup)
│
├── openapi/
│   └── openapi.yaml          Full REST API spec
│
├── Dockerfile
├── .dockerignore
├── go.mod
└── go.sum
```

## Frontend

```
frontend/
├── src/
│   ├── app/
│   │   └── App.tsx           Root component; layout shell, routing, header with action counter
│   │
│   ├── features/
│   │   ├── clusters/
│   │   │   └── ClusterTable.tsx     Cluster inventory table; expandable rows with workload list and live metrics
│   │   ├── recommendations/
│   │   │   └── RecommendationList.tsx   Recommendation cards with filter bar, savings display, action buttons
│   │   ├── actions/
│   │   │   └── ActionHistory.tsx    Completed action log panel
│   │   └── summary/
│   │       └── SummaryCards.tsx     Savings summary cards (total spend, total savings, action count)
│   │
│   ├── services/
│   │   └── api.ts            Typed API client; all fetch calls to the backend
│   │
│   ├── types.ts              Shared TypeScript types matching the backend models
│   └── style.css             Global styles
│
├── index.html
├── vite.config.ts
├── tsconfig.json
├── package.json
├── nginx.conf                Nginx config for Docker: serves built app, proxies /api to backend
├── Dockerfile
└── .dockerignore
```

## Delivery

```
deploy/
└── helm/
    └── smart-cost-optimizer/
        ├── Chart.yaml
        ├── values.yaml               DATA_SOURCE, image tags, resource limits
        └── templates/
            ├── backend-deployment.yaml
            ├── backend-service.yaml
            ├── frontend-deployment.yaml
            └── frontend-service.yaml

scripts/
├── dev.sh                    Bash: installs frontend deps, builds, starts backend and frontend preview
└── dev.ps1                   PowerShell equivalent of dev.sh

docker-compose.yml            Two-container stack: backend (Go/8080) + frontend (Nginx/5173)
Makefile                      Targets: frontend-install, frontend-build, backend-run, run, docker-up, helm-template
```

## Component Connections

```
┌──────────────────────────────────────────────────────────────┐
│  Browser                                                     │
│  React SPA (port 5173)                                       │
│  App.tsx → features/* → services/api.ts                      │
└───────────────────────────┬──────────────────────────────────┘
                            │  HTTP  /api/v1/*
                            ▼
┌──────────────────────────────────────────────────────────────┐
│  Backend (port 8080)                                         │
│  cmd/api/main.go                                             │
│    ├── internal/http/server.go      (Gin routes + handlers)  │
│    ├── internal/recommender/engine  (recommendation logic)   │
│    ├── internal/actions/service     (action execution)       │
│    ├── internal/metrics/provider    (utilization signals)    │
│    ├── internal/pricing/catalog     (cost estimation)        │
│    └── internal/gardener/           (data source)            │
│         ├── client.go   ──────────────────────────────────►  Gardener API (real mode)
│         └── fallback.go             (mock landscape)         │
└──────────────────────────────────────────────────────────────┘
                            │  JSONL write
                            ▼
                    backend/data/actions.jsonl
```

## Data Flow

**On startup:**

1. `config.go` reads env vars
2. Data source is selected (`mock`, `real`, or `auto`)
3. Action history is reloaded from `actions.jsonl`
4. Recommendation engine runs its first pass

**On each recommendation refresh (interval-based):**

1. `gardener/client.go` or `gardener/fallback.go` returns the cluster list
2. `metrics/provider.go` attaches utilization signals to each cluster
3. `pricing/catalog.go` estimates monthly costs
4. `recommender/engine.go` scores each cluster and produces typed recommendations

**On a `POST /api/v1/actions/*` request:**

1. Handler in `http/server.go` validates the request
2. `actions/service.go` executes the action (in-memory mutation or real Gardener API call)
3. The result is appended to `data/actions.jsonl`
4. The recommendation engine re-runs to update the dashboard state
