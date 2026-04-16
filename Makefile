.PHONY: help test test-cover lint lint-fix fmt fmt-check vet build clean deps tidy

# Переменные
GO := go
GOFLAGS := -v
TESTFLAGS := -v -race -coverprofile=coverage.out -covermode=atomic
LINTER := golangci-lint

# По умолчанию выводит справку
help:
	@echo "Доступные команды:"
	@echo "  make test          - запустить тесты"
	@echo "  make test-cover    - запустить тесты с покрытием"
	@echo "  make lint          - запустить линтер"
	@echo "  make lint-fix      - автоматически исправить проблемы линтера"
	@echo "  make fmt           - отформатировать код"
	@echo "  make fmt-check     - проверить форматирование"
	@echo "  make vet           - запустить go vet"
	@echo "  make build         - собрать проект"
	@echo "  make clean         - очистить артефакты сборки"
	@echo "  make deps          - скачать зависимости"
	@echo "  make tidy          - очистить go.mod и go.sum"
	@echo "  make all           - fmt, vet, lint, test"

# Запустить тесты
test:
	$(GO) test $(GOFLAGS) ./...

# Запустить тесты с покрытием
test-cover:
	$(GO) test $(TESTFLAGS) ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Запустить линтер
lint:
	$(LINTER) run

# Автоматически исправить проблемы линтера
lint-fix:
	$(LINTER) run --fix

# Отформатировать код
fmt:
	$(GO) fmt ./...

# Проверить форматирование
fmt-check:
	@test -z $$($(GO) fmt ./...)

# Запустить go vet
vet:
	$(GO) vet ./...

# Собрать проект
build:
	$(GO) build $(GOFLAGS) ./...

# Очистить артефакты сборки
clean:
	rm -f coverage.out coverage.html
	rm -rf bin/
	find . -name "*.test" -delete
	find . -name "*.out" -delete

# Скачать зависимости
deps:
	$(GO) mod download

# Очистить go.mod и go.sum
tidy:
	$(GO) mod tidy

# Запустить все проверки
all: fmt-check vet lint test
