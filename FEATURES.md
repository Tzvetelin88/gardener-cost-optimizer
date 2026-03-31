# Features

## v0.0.2 Features

### Core Platform Features
- Multi-cluster inventory view
- Cluster cost estimation
- Cluster utilization summary
- Savings summary dashboard
- Recommendation engine for cost optimization
- Mock mode for local demos
- Real Gardener mode for integration
- Auto mode to fall back from real to mock
- Configurable idle and utilization thresholds via environment variables (`IDLE_THRESHOLD`, `TARGET_UTILIZATION`)
- Action history persisted to disk and reloaded on restart (`ACTION_LOG_PATH`, default `./data/actions.jsonl`)

### Recommendation Types
- Idle cluster detection
- Hibernation recommendation for underused clusters
- Cheaper placement recommendation for stateless workloads
- Cluster consolidation recommendation for similar underutilized clusters

### Executable Actions
- Hibernate a cluster
- Wake (un-hibernate) a cluster
- Move a stateless workload to a cheaper cluster
- Scale a worker pool

### Dashboard Features
- Cluster inventory table with expand-on-click workload list per cluster
- Per-cluster live metrics (CPU, memory utilization, node count, idle score)
- Wake button for hibernated clusters directly from the inventory table
- Recommendation list with filter bar by kind and risk level
- Risk labels (low / medium / high)
- Estimated monthly savings per recommendation
- Action feedback: loading spinner and inline error message per recommendation card
- Action history panel
- Ready action count in header
- Advisory vs actionable recommendation view

### API Features
- `GET /api/v1/clusters`
- `GET /api/v1/clusters/:name` — workloads and live metrics for a single cluster
- `GET /api/v1/recommendations`
- `GET /api/v1/recommendations/:id`
- `GET /api/v1/actions`
- `GET /api/v1/savings/summary`
- `POST /api/v1/actions/hibernate-cluster`
- `POST /api/v1/actions/wake-cluster`
- `POST /api/v1/actions/scale-nodepool`
- `POST /api/v1/actions/move-workload`
- `GET /healthz`

## Current Limits
- Consolidation is recommendation-only, not executable
- Stateful workload migration is not supported
- Pricing is heuristic, not connected to live cloud billing APIs (AWS Cost Explorer, Azure Cost Management)
- Real workload moves require shoot kubeconfigs configured via `SHOOT_KUBECONFIG_MAP`
- No Prometheus integration; utilization is derived from Kubernetes resource requests, not actual usage metrics
- `OptimizationPolicy` CRD type is defined but no controller processes it yet

## Short Summary
This release supports:
- seeing all clusters with per-cluster workload detail
- seeing savings opportunities filtered by kind and risk
- hibernating idle clusters and waking them back up
- moving stateless workloads to cheaper clusters
- recommending consolidation
- tracking completed actions across restarts
- configuring recommendation sensitivity via environment variables
