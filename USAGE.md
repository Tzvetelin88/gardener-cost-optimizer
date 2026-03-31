# Smart Cost Optimizer Usage

This document explains the product through examples.

The goal is that someone reading these scenarios can immediately understand:

- what Smart Cost Optimizer looks at
- what problems it solves
- how the API and dashboard are used
- what outcome an operator gets

## What The Product Does

Smart Cost Optimizer sits on top of Gardener and Kubernetes and helps platform teams make better cost decisions across clusters.

It does four things:

1. Reads cluster inventory.
2. Reads or simulates workload and utilization signals.
3. Produces recommendations with estimated savings.
4. Lets operators execute safe manual actions.

This is especially useful because Gardener already handles cluster lifecycle and hibernation, but it does not itself provide an intelligent global optimizer for cross-cluster decisions like workload moves or consolidation recommendations.

## Mental Model

Think of the product like this:

- Gardener manages Kubernetes clusters
- Kubernetes runs workloads inside those clusters
- Smart Cost Optimizer analyzes the landscape above both of them

The output is not just metrics. The output is:

- recommendations
- estimated savings
- action suggestions
- operator-triggered changes

## Example 1: Idle Dev Cluster Hibernation

### Situation

You have a non-production cluster called `garden-dev/dev-aws-b`.

It has:

- low utilization
- no important active workloads
- a monthly infrastructure cost that is still being paid

### What The Product Shows

The dashboard lists the cluster in inventory and shows a recommendation such as:

- kind: `idle-cluster`
- reason: cluster is mostly idle
- savings: estimated monthly savings if hibernated
- risk: low
- action: `hibernate-cluster`

### API Example

Read recommendations:

```bash
curl http://localhost:8080/api/v1/recommendations
```

Run the hibernation action:

```bash
curl -X POST http://localhost:8080/api/v1/actions/hibernate-cluster \
  -H "Content-Type: application/json" \
  -d '{"clusterName":"garden-dev/dev-aws-b"}'
```

### Outcome

The operator sees:

- the recommendation
- the expected savings
- a completed action in history

In:

- `mock` mode, the cluster state is updated in memory
- `real` mode, the request goes through Gardener and updates the `Shoot`

This example shows that the product is not just a dashboard. It can turn an optimization suggestion into an operator action.

## Example 2: Move Stateless Workload To Cheaper Cluster

### Situation

A stateless workload such as `web/catalog-api` is currently running on a more expensive dev cluster.

Another cluster in the same region and purpose boundary is cheaper and has enough free capacity.

### What The Product Checks

The optimizer compares:

- source cluster monthly cost
- target cluster monthly cost
- utilization on target cluster
- workload type
- isolation and relevance constraints

The product only recommends automatic move actions for safe stateless workloads in this MVP.

### What The Product Shows

Recommendation example:

- kind: `cheaper-placement`
- subject: `web/catalog-api`
- source: more expensive cluster
- target: cheaper cluster
- evidence: stateless workload, cheaper target, enough capacity
- savings: estimated monthly savings
- risk: medium

### API Example

Read recommendations:

```bash
curl http://localhost:8080/api/v1/recommendations
```

Execute move:

```bash
curl -X POST http://localhost:8080/api/v1/actions/move-workload \
  -H "Content-Type: application/json" \
  -d '{
    "sourceCluster":"garden-dev/dev-aws-a",
    "targetCluster":"garden-dev/dev-aws-b",
    "namespace":"web",
    "workloadName":"catalog-api"
  }'
```

### Outcome

The operator gets:

- a visible record in action history
- lower placement cost for that workload
- a clear explanation of why the move is suggested

In:

- `mock` mode, the workload is moved in the mock landscape
- `real` mode, the system clones the deployment to the target shoot and scales the source down

This example shows the cross-cluster optimizer story very clearly:

"Run the same workload in a cheaper but still relevant place."

## Example 3: Cluster Consolidation Recommendation

### Situation

Two clusters serve a similar function:

- same cloud
- same region
- same purpose
- both underutilized

For example:

- `dev-aws-a`
- `dev-aws-b`

### What The Product Shows

The optimizer creates a recommendation like:

- kind: `cluster-consolidation`
- subject: `dev-aws-a + dev-aws-b`
- evidence:
  - same region
  - same purpose
  - both low utilization
  - combined load is still manageable
- savings: estimated savings from running one less cluster
- risk: high
- executable: false

### Why It Matters

This is important because not every recommendation should be turned into an immediate automatic action.

Consolidation is usually more complex because it may involve:

- isolation concerns
- team ownership
- deployment dependencies
- data or service coupling

### Outcome

The operator gets a decision-support recommendation rather than a blind automation.

This example shows that the product is not only about "do action now". It is also about giving a platform team better operational intelligence.

## Example 4: Oversized Worker Pool

### Situation

A cluster has a worker pool sized for older demand, but current workload usage is much lower.

### What The Product Shows

The optimizer can recommend:

- lower minimum node count
- lower maximum node count
- estimated savings from reduced worker capacity

### API Example

```bash
curl -X POST http://localhost:8080/api/v1/actions/scale-nodepool \
  -H "Content-Type: application/json" \
  -d '{
    "clusterName":"garden-dev/dev-aws-a",
    "workerPool":"system",
    "minimum":2,
    "maximum":3
  }'
```

### Outcome

The operator reduces the size of the worker pool in a controlled way.

In:

- `mock` mode, the pool is resized in memory
- `real` mode, the Gardener `Shoot` worker configuration is updated

This example shows that the product can optimize not only placement and cluster state, but also cluster shape.

## Example 5: Dashboard Summary For Leadership

### Situation

A platform lead wants a quick answer to:

- how many clusters do we run
- where are the biggest savings opportunities
- how many actions are ready now

### What The Product Shows

The dashboard summary provides:

- total monthly spend
- total monthly savings opportunity
- actionable item count
- advisory item count

### API Example

```bash
curl http://localhost:8080/api/v1/savings/summary
```

Example response shape:

```json
{
  "totalMonthlySpend": 2100,
  "totalMonthlySavings": 540,
  "actionableCount": 3,
  "advisoryCount": 1
}
```

### Outcome

This gives stakeholders a high-level answer without needing to inspect raw Kubernetes or Gardener objects.

This example shows that the product is useful not only for cluster operators, but also for engineering leads and platform owners.

## Example 6: Mock Mode Demo

### Situation

You do not yet have a Gardener environment but want to demonstrate the product.

### How To Run

With Docker:

```bash
docker compose up --build
```

With shell script:

```bash
DATA_SOURCE=mock sh ./scripts/dev.sh
```

With PowerShell:

```powershell
.\scripts\dev.ps1 -DataSource mock
```

### What Happens

The system uses:

- built-in clusters
- built-in workloads
- built-in recommendations
- simulated actions

### Outcome

This is perfect for:

- demos
- UI development
- API development
- explaining the product to stakeholders before real integration exists

## Example 7: Real Gardener Integration

### Situation

You now have access to a real Gardener landscape and want real inventory and real actions.

### How To Run

PowerShell example:

```powershell
$env:DATA_SOURCE="real"
$env:GARDENER_KUBECONFIG="C:\kubeconfigs\garden.yaml"
$env:GARDENER_CONTEXT="garden-admin"
$env:SHOOT_KUBECONFIG_MAP="dev-aws-a=C:\kubeconfigs\dev-aws-a.yaml,dev-aws-b=C:\kubeconfigs\dev-aws-b.yaml"
cd .\backend
go run ./cmd/api
```

Shell example:

```bash
DATA_SOURCE=real \
GARDENER_KUBECONFIG=/path/to/garden.yaml \
GARDENER_CONTEXT=garden-admin \
SHOOT_KUBECONFIG_MAP="dev-aws-a=/kube/dev-aws-a.yaml,dev-aws-b=/kube/dev-aws-b.yaml" \
go run ./backend/cmd/api
```

### What Happens

The product will:

- read real Gardener `Shoot` objects
- inspect real cluster placement data
- use real shoot access for workload actions
- expose the same API and dashboard, but now against real systems

### Outcome

The same operator workflow used in mock mode now becomes operational in a real Gardener-based environment.

This shows the product transition very clearly:

- first as demo and design tool
- then as real platform capability

## Example 8: Auto Mode

### Situation

You want one startup mode that prefers real Gardener, but still works on a laptop without it.

### How To Run

```bash
DATA_SOURCE=auto sh ./scripts/dev.sh
```

### What Happens

The backend:

1. tries to connect to Gardener
2. if successful, uses real mode
3. if not, falls back to mock mode

### Outcome

This is useful for:

- developer onboarding
- demos in mixed environments
- gradual migration from local mode to integrated mode

## Quick Demo Script

If you want to explain the product live in 2 minutes, this is a good flow:

1. Open the dashboard.
2. Show cluster inventory.
3. Open recommendations.
4. Click an idle cluster recommendation.
5. Show estimated savings.
6. Trigger hibernation.
7. Open action history.
8. Explain that the same flow works with real Gardener in `real` mode.

## Short Value Statement

If someone asks, "What does this product actually do?", the shortest clear answer is:

Smart Cost Optimizer uses Gardener and Kubernetes cluster data to identify wasted spend, recommend better placement and consolidation decisions, and let operators safely execute cost-saving actions through an API and dashboard.
