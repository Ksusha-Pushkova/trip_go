APP_NAME := trip-service
BIN_DIR := bin
.DEFAULT_GOAL := help

.PHONY: help
help: ## показывает список целй
	@echo "available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'


.PHONY: build
build: ## собирает бинарь в bin/
	go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/trip-service 


.PHONY: run
run: ## запускает сервис
	set -a && source .env && set +a && go run .cmd/trip-service


.PHONY: generate
generate: ## генрирует код из OpenAPI
	go tool oapi-codegen \
	-generate types,chi-server \
	-package api \
	-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
	-o api/api.gen.go \
	contracts/openapi/trip-service.openapi.yaml


.PHONY: migrate
migrate: ## накатить все миграции
	set -a && source .env && set +a && go tool goose -dir migrations postgres "$$DATABASE_URL" up


.PHONY: migrate-down
migrate-down: ## откатить последнюю миграцию
	set -a && source .env && set +a && go tool goose -dir migrations postgres "$$DATABASE_URL" down


.PHONY: test
test: ## прогнать все тесты
	go test -race ./...


.PHONY: clean
clean: ## удаляет собранные бинари
	-rm -rf $(BIN_DIR)