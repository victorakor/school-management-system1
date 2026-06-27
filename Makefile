.PHONY: run build migrate test tailwind-watch tailwind-build docker-up docker-down lint vet

# ─── Go ───────────────────────────────────────────────────
run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/server

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test -v -race ./...

# ─── Database ─────────────────────────────────────────────
migrate:
	go run ./cmd/server -migrate-only

# ─── Tailwind CSS ─────────────────────────────────────────
tailwind-watch:
	npx tailwindcss -i ./web/static/css/tailwind.css -o ./web/static/css/app.css --watch

tailwind-build:
	npx tailwindcss -i ./web/static/css/tailwind.css -o ./web/static/css/app.css --minify

# ─── Docker (local dev) ───────────────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# ─── Dev (full stack) ─────────────────────────────────────
dev: docker-up
	@echo "Waiting for services..."
	@sleep 2
	@$(MAKE) run
