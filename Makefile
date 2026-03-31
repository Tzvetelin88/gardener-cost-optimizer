SHELL := /bin/sh

.PHONY: frontend-install frontend-build backend-run frontend-run api run run-ps docker-up helm-template

frontend-install:
	cd frontend && npm install

frontend-build:
	cd frontend && npm run build

backend-run:
	cd backend && go run ./cmd/api

frontend-run:
	cd frontend && VITE_API_BASE_URL=http://localhost:8080/api/v1 npm run build && npm run preview -- --host=0.0.0.0 --port=4173

api:
	cd backend && go run ./cmd/api

run:
	sh ./scripts/dev.sh

run-ps:
	powershell -ExecutionPolicy Bypass -File ./scripts/dev.ps1

docker-up:
	docker compose up --build

helm-template:
	helm template smart-cost-optimizer ./deploy/helm/smart-cost-optimizer
