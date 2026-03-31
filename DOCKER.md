# Docker Setup

Docker Compose is the easiest way to run the full stack locally. It requires no local Go or Node installation and works consistently on Windows, Linux, and macOS.

## Container Files

```
docker-compose.yml
backend/Dockerfile
backend/.dockerignore
frontend/Dockerfile
frontend/.dockerignore
frontend/nginx.conf
```

## What Runs

Two containers start:

| Container | Description | Port |
|---|---|---|
| `gardener-cost-optimizer-api` | Go backend in `mock` mode | `8080` |
| `gardener-cost-optimizer-frontend` | Built React app served by Nginx | `5173` |

The frontend Nginx config proxies all `/api` requests to the backend container, so the browser only needs to reach `localhost:5173`.

## Commands

Build and start:

```bash
docker compose up --build
```

Start in background:

```bash
docker compose up --build -d
```

Stop:

```bash
docker compose down
```

## Endpoints After Startup

```
http://localhost:5173                             frontend dashboard
http://localhost:8080/healthz                     backend health → "ok"
http://localhost:8080/api/v1/recommendations      recommendations
http://localhost:8080/api/v1/savings/summary      savings summary
```

## Default Mode

The Docker setup defaults to `DATA_SOURCE=mock`. The full dashboard, API, and action flow work without any Gardener environment.

## Running Against Real Gardener

Override the backend environment to connect to a real Gardener landscape:

```bash
docker compose run --rm \
  -e DATA_SOURCE=real \
  -e GARDENER_KUBECONFIG=/kubeconfigs/garden.yaml \
  -e GARDENER_CONTEXT=garden-admin \
  smart-cost-optimizer-api
```

For a production deployment, use the Helm chart instead of Docker Compose. See `deploy/helm/smart-cost-optimizer`.

## Quick Verification

After `docker compose up --build`, run these to confirm the stack is working:

```bash
curl http://localhost:8080/healthz
# → ok

curl http://localhost:8080/api/v1/recommendations
# → JSON array of recommendations

curl http://localhost:8080/api/v1/savings/summary
# → { "totalMonthlySpend": ..., "totalMonthlySavings": ..., ... }
```
