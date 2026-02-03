# 宝宝时光轴后端 (Anan Diary Backend)

基于 Go + Gin + GORM 的后端服务。

## 项目结构

```
ad-backend/
├── cmd/server/          # 主程序入口
│   └── main.go         # 服务器启动逻辑
├── internal/
│   ├── models/         # 数据模型
│   │   ├── user.go
│   │   ├── baby.go
│   │   ├── family_member.go
│   │   ├── growth_record.go
│   │   └── milestone.go
│   ├── repository/     # 数据访问层
│   │   ├── user_repo.go
│   │   ├── baby_repo.go
│   │   ├── family_member_repo.go
│   │   ├── growth_record_repo.go
│   │   └── milestone_repo.go
│   └── handlers/       # 请求处理器
│       ├── auth.go
│       ├── baby.go
│       ├── family_member.go
│       ├── growth_record.go
│       └── milestone.go
├── pkg/
│   ├── database/       # 数据库配置
│   │   └── database.go
│   ├── middleware/     # 中间件
│   │   └── auth.go
│   └── response/       # 响应封装
│       └── response.go
├── migrations/         # 数据库迁移
│   └── init.sql
├── Makefile            # 构建脚本
├── go.mod              # Go模块依赖
└── .env.example        # 环境变量示例
```

## 快速开始

### 环境准备

- Go 1.21+
- PostgreSQL 14+

### 安装

```bash
# 下载依赖
go mod download

# 配置数据库
# 编辑 migrations/init.sql 设置数据库连接信息
```

### 运行

```bash
# 编译
make build

# 运行
make run

# 或直接运行
go run cmd/server/main.go
```

服务将在 `http://localhost:8080` 启动。

### 数据库初始化

```bash
# 使用 psql 执行迁移脚本
psql -h localhost -U postgres -f migrations/init.sql
```

## API端点

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | `/api/auth/register` | 用户注册 | 否 |
| POST | `/api/auth/login` | 用户登录 | 否 |
| GET | `/api/baby/current` | 获取当前宝宝 | 是 |
| GET | `/api/baby` | 获取宝宝列表 | 是 |
| POST | `/api/baby` | 创建宝宝档案 | 是 |
| GET | `/api/baby/:id` | 获取宝宝详情 | 是 |
| DELETE | `/api/baby/:id` | 删除宝宝档案 | 是 |
| GET | `/api/growth-record/timeline` | 获取时光轴 | 是 |
| POST | `/api/growth-record` | 创建成长记录 | 是 |
| DELETE | `/api/growth-record/:id` | 删除记录 | 是 |
| GET | `/api/family-member/:id` | 获取成员列表 | 是 |
| POST | `/api/family-member` | 添加成员 | 是 |
| GET | `/api/milestone` | 获取里程碑标签 | 是 |
| GET | `/api/milestone/check` | 检查里程碑 | 是 |
| GET | `/health` | 健康检查 | 否 |

## 数据库模型

### Users
```go
type User struct {
    ID           uint      `gorm:"primaryKey"`
    Email        string    `gorm:"uniqueIndex;not null"`
    PasswordHash string    `gorm:"not null"`
    CreatedAt    time.Time
}
```

### Babies
```go
type Baby struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"not null;index"`
    Name      string    `gorm:"size:100;not null"`
    BirthDate string    `gorm:"size:10;not null"`
    BabyID    string    `gorm:"uniqueIndex;size:50;not null"`
    CreatedAt time.Time
}
```

### GrowthRecords
```go
type GrowthRecord struct {
    ID        uint      `gorm:"primaryKey"`
    BabyID    uint      `gorm:"not null;index"`
    Content   string    `gorm:"type:text;not null"`
    Tags      string    `gorm:"type:text"`
    CreatedAt time.Time
}
```

## 构建工具

```bash
make build     # 编译项目
make run       # 编译并运行
make test      # 运行测试
make clean     # 清理编译文件
make fmt       # 格式化代码
make deps      # 依赖检查
```

## 环境变量

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=anan_diary
JWT_SECRET=your-secret-key
```
