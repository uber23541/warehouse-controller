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

## install-tools: установить golangci-lint и swag нужных версий
.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest

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

## test: прогнать тесты с детектором гонок
.PHONY: test
test:
	go test ./... -race -count=1

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
