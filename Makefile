.PHONY: test validate build run web-dev up sandbox-up sandbox-down
test:
	go vet ./... && go test ./... && cd web && npm test && npm run typecheck
validate:
	go run ./cmd/validate catalog
build:
	CGO_ENABLED=0 go build -o bin/ ./cmd/...
run:
	CATALOG_DIR=catalog go run ./cmd/gateway
web-dev:
	cd web && npm install && npm run dev
up:
	docker compose up --build
sandbox-up:
	docker compose --profile sandbox up --build
sandbox-down:
	docker compose --profile sandbox down
