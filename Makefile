.PHONY: web api ai up down

web:
	cd apps/web && npm run dev

api:
	cd services/api && go run ./cmd/server

ai:
	cd services/ai && uvicorn app.main:app --reload --port 5000

up:
	docker compose -f infra/docker/docker-compose.yml up --build

down:
	docker compose -f infra/docker/docker-compose.yml down
