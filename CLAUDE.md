# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 快速开发

### 命令
```bash
make run          # 编译并运行
make migrate      # 初始化数据库
make test         # 运行测试
make clean        # 清理编译文件
make fmt          # 格式化代码
```

### 数据库迁移
- 迁移脚本位于 `migrations/*.sql`
- 使用 `InitDBWithMigration` 函数执行完整迁移
- 当前数据库结构与 models 有差异，以 SQL 迁移脚本为准（如 `birth_date` 为 TIMESTAMP）

## 关键架构

### 分层结构
```
handlers/   - HTTP 请求处理层
repository/ - 数据访问层 (GORM)
models/     - 数据模型
```

### 中间件
- `AuthMiddleware()` - JWT 认证
- `Response` - 统一响应格式

### JWT Secret
硬编码在 `pkg/middleware/auth.go:14`，生产环境需要修改。

## 数据模型关系

```
users (role: 爸爸/妈妈/爷爷/奶奶/姥姥/姥爷/哥哥/姐姐)
  ↓ user_id
family_members (baby_id, role, name) - UNIQUE(baby_id, role)
  ↓ baby_id
babies (user_id, name, birth_date)
  ↓ baby_id
growth_records (baby_id, content, tags)
```

**注意**: `babies` 表的 `user_id` 是创建者，`family_members` 表包含所有家庭成员。
