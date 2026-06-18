# Makefile для warehouse-controller.
# Требует `make` (на Windows — через Git Bash или WSL).
# Версии инструментов совпадают с CI (.github/workflows/ci.yml).

# Параметры приложения
BINARY     := warehouse-controller
CMD_PATH   := ./cmd
SWAG_ENTRY := cmd/main.go
DOCS_DIR   := docs

.DEFAULT_GOAL := help

## help: показать список целей
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -E 's/## //'

## install-tools: установить golangci-lint, swag и mockery нужных версий
.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/vektra/mockery/v2@v2.53.3

## mocks: сгенерировать моки по .mockery.yaml
.PHONY: mocks
mocks:
	mockery

## fmt: отформатировать код (gofmt + goimports через golangci-lint)
.PHONY: fmt
fmt:
	golangci-lint fmt

## fmt-check: проверить форматирование без правок (как в CI)
.PHONY: fmt-check
fmt-check:
	golangci-lint fmt --diff

## vet: запустить go vet
.PHONY: vet
vet:
	go vet ./...

## lint: запустить golangci-lint
.PHONY: lint
lint:
	golangci-lint run ./...

## test: прогнать юнит-тесты с детектором гонок
.PHONY: test
test:
	go test ./test/unit/... -race -count=1

## test-integration: прогнать интеграционные тесты (требуется Docker)
.PHONY: test-integration
test-integration:
	go test -tags=integration ./test/integration/... -count=1

## cover: посчитать покрытие юнит-тестами (итог в строке total:)
.PHONY: cover
cover:
	go test ./test/unit/... -coverpkg=./internal/... -coverprofile=cover.out -count=1
	go tool cover -func=cover.out

## cover-html: открыть HTML-отчёт покрытия
.PHONY: cover-html
cover-html:
	go test ./test/unit/... -coverpkg=./internal/... -coverprofile=cover.out -count=1
	go tool cover -html=cover.out

## tidy: привести go.mod/go.sum в порядок
.PHONY: tidy
tidy:
	go mod tidy

## swagger: сгенерировать swagger-документацию в docs/
.PHONY: swagger
swagger:
	swag init -g $(SWAG_ENTRY) -o $(DOCS_DIR)

## build: собрать бинарник в bin/
.PHONY: build
build:
	go build -o bin/$(BINARY) $(CMD_PATH)

## run: запустить приложение локально
.PHONY: run
run:
	go run $(CMD_PATH)

## check: локальная имитация CI (форматирование + vet + lint + тесты)
.PHONY: check
check: fmt-check vet lint test

## up: поднять стек через docker compose
.PHONY: up
up:
	docker compose up -d --build

## down: остановить стек
.PHONY: down
down:
	docker compose down

## logs: смотреть логи приложения
.PHONY: logs
logs:
	docker compose logs -f app

## docker-build: собрать docker-образ локально
.PHONY: docker-build
docker-build:
	docker build -t $(BINARY):local .

## clean: удалить артефакты сборки
.PHONY: clean
clean:
	rm -rf bin/

# --- Kubernetes (minikube) ---
K8S_DIR := deploy/k8s
K8S_NS  := warehouse

## k8s-deps: добавить Helm-репо Bitnami и установить Postgres + Redis
.PHONY: k8s-deps
k8s-deps:
	helm repo add bitnami https://charts.bitnami.com/bitnami
	helm repo update
	kubectl apply -f $(K8S_DIR)/namespace.yaml
	helm upgrade --install warehouse-postgresql bitnami/postgresql -n $(K8S_NS) -f $(K8S_DIR)/helm/postgres-values.yaml
	helm upgrade --install warehouse-redis bitnami/redis -n $(K8S_NS) -f $(K8S_DIR)/helm/redis-values.yaml

## k8s-up: применить манифесты приложения (configmap, deployment, service, ingress)
.PHONY: k8s-up
k8s-up:
	kubectl apply -k $(K8S_DIR)

## k8s-down: удалить ресурсы приложения и Helm-релизы
.PHONY: k8s-down
k8s-down:
	kubectl delete -k $(K8S_DIR) --ignore-not-found
	helm uninstall warehouse-postgresql warehouse-redis -n $(K8S_NS) || true
