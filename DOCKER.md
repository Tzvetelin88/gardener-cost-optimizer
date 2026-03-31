# Docker Usage

This project includes a containerized setup so it can run consistently on Windows, Linux, and macOS without relying on local Go or Node installations.

## Files

Container-related files in the repository root:

- `docker-compose.yml`
- `backend/Dockerfile`
- `backend/.dockerignore`
- `frontend/Dockerfile`
- `frontend/.dockerignore`
- `frontend/nginx.conf`

## What Docker Runs

The compose stack starts two containers:

- `gardener-cost-optimizer-api`
  - Go backend
  - listens on port `8080`
  - defaults to `mock` mode in Docker
- `gardener-cost-optimizer-frontend`
  - built React app served by Nginx
  - listens on port `5173` on the host
  - proxies `/api` to the backend container

## Run With Docker Compose

From the repository root:

```bash
docker compose up --build
```

Run in background:

```bash
docker compose up --build -d
```

Stop the stack:

```bash
docker compose down
```

## Default Docker Endpoints

After startup:

- frontend: `http://localhost:5173`
- backend health: `http://localhost:8080/healthz`
- backend API: `http://localhost:8080/api/v1/recommendations`

## Default Mode

The Docker setup uses:

- `DATA_SOURCE=mock`

This makes the container stack work without a real Gardener landscape.

## Run Docker In Mock Mode

The default compose file already uses mock mode:

```yaml
environment:
  DATA_SOURCE: mock
```

This is the easiest way to demo the project.

## Run Docker In Real Mode

If you want to connect to a real Gardener environment, override the backend environment values.

Example:

```bash
docker compose run --rm \
  -e DATA_SOURCE=real \
  -e GARDENER_KUBECONFIG=/kubeconfigs/garden.yaml \
  -e GARDENER_CONTEXT=garden-admin \
  smart-cost-optimizer-api
```

For a real deployment, it is usually better to use the Helm chart instead of local Docker Compose.

## Health Check

The backend container exposes:

```text
http://localhost:8080/healthz
```

Expected result:

```text
ok
```

## Example Verification

Check recommendations:

```bash
curl http://localhost:8080/api/v1/recommendations
```

Check summary:

```bash
curl http://localhost:8080/api/v1/savings/summary
```

## Notes

- Docker Compose is the most consistent cross-platform way to run this project locally.
- The shell and PowerShell scripts are still useful, but Docker avoids local runtime differences.
- The Docker setup is best for `mock` mode demos and quick validation.
