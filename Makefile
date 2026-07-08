.PHONY: build run migrate test vet fmt clean help

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
	@echo "  make build     - 编译主项目"
	@echo "  make run       - 编译并运行主项目"
	@echo "  make vet       - 静态检查"
	@echo "  make test      - 运行测试"
	@echo "  make migrate   - 初始化主数据库"
	@echo "  make fmt       - 格式化代码"
	@echo "  make clean     - 清理编译文件"

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

# 静态检查
vet:
	@echo "静态检查..."
	go vet ./cmd/server ./internal/... ./pkg/...

# 运行测试
test:
	@echo "运行测试..."
	go test ./internal/... ./pkg/... ./models/...

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