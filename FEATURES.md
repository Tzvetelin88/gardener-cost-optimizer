# Features

## Supported Features

### Core Platform Features
- Multi-cluster inventory view
- Cluster cost estimation
- Cluster utilization summary
- Savings summary dashboard
- Recommendation engine for cost optimization
- Mock mode for local demos
- Real Gardener mode for integration
- Auto mode to fall back from real to mock

### Recommendation Types
- Idle cluster detection
- Hibernation recommendation for underused clusters
- Cheaper placement recommendation for stateless workloads
- Cluster consolidation recommendation for similar underutilized clusters

### Executable Actions
- Hibernate a cluster
- Move a stateless workload to a cheaper cluster
- Scale a worker pool

### Dashboard Features
- Cluster inventory table
- Recommendation list
- Risk labels
- Estimated monthly savings
- Action history
- Ready action count
- Advisory vs actionable recommendation view

### API Features
- `GET /api/v1/clusters`
- `GET /api/v1/recommendations`
- `GET /api/v1/recommendations/:id`
- `GET /api/v1/actions`
- `GET /api/v1/savings/summary`
- `POST /api/v1/actions/hibernate-cluster`
- `POST /api/v1/actions/scale-nodepool`
- `POST /api/v1/actions/move-workload`
- `GET /healthz`

## Current MVP Limits
- Consolidation is recommendation-only, not executable
- Stateful workload migration is not supported
- Pricing is heuristic, not connected to cloud billing APIs
- Real workload moves require shoot kubeconfigs
- Full production-grade dependency migration is only partial in this MVP

## Short Summary
This MVP supports:
- seeing clusters
- seeing savings opportunities
- hibernating idle clusters
- moving stateless workloads to cheaper clusters
- recommending consolidation
- tracking completed actions
