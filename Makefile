.PHONY: infra-up infra-down migrate-up migrate-down backend-test integration-test smoke-rag backend-run worker-run

infra-up:
	docker compose up -d mysql redis minio qdrant

infra-down:
	docker compose down

migrate-up:
	docker compose --profile tools run --rm migrate

migrate-down:
	docker compose --profile tools run --rm migrate -path=/migrations -database="mysql://whatsnext:$${MYSQL_PASSWORD:-whatsnext_dev}@tcp(mysql:3307)/whatsnext" down 1

backend-test:
	cd backend && go test ./...

integration-test:
	cd backend && go test -tags=integration ./tests/integration

smoke-rag:
	./scripts/smoke-rag.sh

backend-run:
	cd backend && go run ./cmd/api

worker-run:
	cd backend && go run ./cmd/worker
