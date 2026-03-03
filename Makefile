.PHONY: build run migrate test test-db clean

# 变量定义
APP_NAME=anan-diary-backend
DB_NAME=anan_diary
DB_USER=postgres
DB_PORT=5432

# 默认目标
.DEFAULT_GOAL := help

# 帮助信息
help:
	@echo "可用命令:"
	@echo "  make build        - 编译主项目"
	@echo "  make run          - 运行主项目"
	@echo "  make migrate      - 初始化主数据库"
	@echo "  make test-db      - 运行测试数据库和场景"
	@echo "  make test-db-run  - 启动测试服务器"
	@echo "  make clean        - 清理编译文件"

# 编译主项目
build:
	@echo "编译主项目..."
	go build -o bin/$(APP_NAME) ./cmd/server

# 运行主项目
run: build
	@echo "运行主项目..."
	go run ./cmd/server

# 初始化主数据库
migrate:
	@echo "初始化主数据库..."
	psql -h localhost -p $(DB_PORT) -U $(DB_USER) -f migrations/init.sql

# 编译测试服务器
test-build:
	@echo "编译测试服务器..."
	go build -o bin/test_server ./cmd/test/main.go

# 运行测试数据库和场景
test-db: test-build
	@echo "启动测试服务器..."
	@echo "测试服务器将在 http://localhost:8081 运行"
	@echo "测试场景将在 10 秒后自动运行..."
	@sleep 10 && go run ./cmd/test/run_test.go

# 启动测试服务器（后台运行）
test-db-run:
	@echo "启动测试服务器..."
	@go run ./cmd/test/main.go &

# 清理编译文件
clean:
	@echo "清理编译文件..."
	rm -rf bin/
	go clean

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 依赖检查
deps:
	@echo "检查依赖..."
	go mod tidy
