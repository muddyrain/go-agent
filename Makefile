APP_NAME := agenthub
MAIN_PACKAGE := ./cmd/agenthub
BUILD_DIR := bin
BINARY := $(BUILD_DIR)/$(APP_NAME)
GO := go
GOTESTSUM := gotestsum
COVERAGE_FILE := coverage.out

.DEFAULT_GOAL := help

.PHONY: help run fmt test test-pretty coverage vet build check clean

help:
	@printf "可用命令：\n"
	@printf "  make run    运行 AgentHub\n"
	@printf "  make fmt    格式化 Go 代码\n"
	@printf "  make test       运行全部测试\n"
	@printf "  make test-pretty 使用彩色格式运行全部测试（需要 gotestsum）\n"
	@printf "  make coverage   生成测试覆盖率报告\n"
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

test-pretty:
	@command -v $(GOTESTSUM) >/dev/null 2>&1 || { \
		printf "未安装 gotestsum，请先运行：go install gotest.tools/gotestsum@latest\n"; \
		exit 1; \
	}
	$(GOTESTSUM) --format testname -- ./...

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