APP_NAME := agenthub
MAIN_PACKAGE := ./cmd/agenthub
BUILD_DIR := bin
BINARY := $(BUILD_DIR)/$(APP_NAME)
GO := go
COVERAGE_FILE := coverage.out

.DEFAULT_GOAL := help

.PHONY: help run fmt test coverage vet build check clean

help:
	@printf "可用命令：\n"
	@printf "  make run    运行 AgentHub\n"
	@printf "  make fmt    格式化 Go 代码\n"
	@printf "  make test   运行全部测试\n"
	@printf "  make coverage  生成测试覆盖率报告\n"
	@printf "  make vet    执行静态检查\n"
	@printf "  make build  编译 AgentHub\n"
	@printf "  make check  执行完整项目检查\n"
	@printf "  make clean  删除构建产物\n"

run:
	$(GO) run $(MAIN_PACKAGE)

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

coverage:
	$(GO) test -count=1 -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -func=$(COVERAGE_FILE)

vet:
	$(GO) vet ./...

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BINARY) $(MAIN_PACKAGE)

check: fmt test vet build

clean:
	rm -rf $(BUILD_DIR)