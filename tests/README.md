# 宝宝时光轴 - 家庭成员功能测试说明

## 概述

本测试项目验证了家庭成员功能的完整流程，包括注册、登录、宝宝档案关联、记录共享等功能。

## 测试环境

- **测试数据库**: `anan_diary_test`
- **测试服务器端口**: `8081`
- **主服务器端口**: `8080`

## 测试场景

### 场景1: 妈妈注册，创建宝宝A的档案

**步骤**:
1. 妈妈使用邮箱 `mom@example.com`、密码 `mom123` 注册
2. 选择角色: 妈妈
3. 填写出生证编号: `BABY001`

**预期结果**:
- 注册成功
- 自动创建宝宝档案
- baby_id 已关联

### 场景2: 爸爸注册，关联妈妈的宝宝

**步骤**:
1. 爸爸使用邮箱 `dad@example.com`、密码 `dad123` 注册
2. 选择角色: 爸爸
3. 填写出生证编号: `BABY001` (与妈妈相同)

**预期结果**:
- 注册成功
- 自动关联妈妈的宝宝档案 (baby_id 已存在)
- 不创建新的宝宝档案
- 返回相同的 baby_id

### 场景3: 爸爸登录后点击取消，展示妈妈的记录

**步骤**:
1. 爸爸使用 `dad@example.com` / `dad123` 登录
2. 点击"取消"按钮

**预期结果**:
- 登录成功
- baby_id 已存在于 localStorage
- 显示时光轴（包含妈妈的记录）

### 场景4: 爸爸新增一条宝宝A的记录

**步骤**:
1. 爸爸新增成长记录
2. 内容: "爸爸记录 - [当前日期]"

**预期结果**:
- 记录创建成功
- 时光轴有2条记录（妈妈的1条 + 爸爸的1条）
- 两个家庭成员都能看到对方添加的记录

### 场景5: 爸爸创建宝宝B的档案

**步骤**:
1. 爸爸使用相同账号 `dad@example.com` / `dad123` 注册
2. 选择角色: 爸爸
3. 填写出生证编号: `BABY002` (不同的宝宝)

**预期结果**:
- 注册成功
- 创建新的宝宝档案
- 宝宝列表显示2个宝宝（宝宝A和宝宝B）
- 每个宝宝的记录是独立的

### 场景6: 爸爸新增一条宝宝B的记录

**步骤**:
1. 爸爸新增宝宝B的成长记录

**预期结果**:
- 记录创建成功
- 宝宝B的时光轴有1条记录

### 场景7: 妈妈登录后查看宝宝的记录

**步骤**:
1. 妈妈使用 `mom@example.com` / `mom123` 登录
2. 查看宝宝列表和时光轴

**预期结果**:
- 登录成功
- 可以看到关联的宝宝列表
- 可以看到宝宝的时光轴记录

## 运行测试

### 方法1: 使用 Make 命令（推荐）

```bash
cd ad-backend
make test-db
```

这将自动:
1. 编译测试服务器
2. 启动测试服务器
3. 等待10秒后自动运行测试场景

### 方法2: 手动运行

1. 启动测试服务器:
```bash
cd ad-backend
make test-db-run
```

2. 在另一个终端运行测试:
```bash
cd ad-backend
go run ./cmd/test/run_test.go
```

## 测试输出示例

```
╔══════════════════════════════════════════════════════════╗
║          宝宝时光轴 - 家庭成员测试脚本                  ║
╚══════════════════════════════════════════════════════════╝

【场景1】妈妈注册，创建宝宝A的档案
───────────────────────────────────────────────────────────
✓ 妈妈注册成功 (mom@example.com)
✓ 宝宝档案创建成功 (baby_id=1, baby_id=BABY001)
  ├─ 用户名: 李妈妈
  ├─ 角色: 妈妈
  └─ 注册信息: 已关联宝宝档案

【场景2】爸爸注册，关联妈妈的宝宝
───────────────────────────────────────────────────────────
✓ 爸爸注册成功 (dad@example.com)
✓ 成功关联妈妈的宝宝档案 (baby_id=1)
  ├─ 用户名: 张爸爸
  └─ 注册信息: 自动关联已存在的宝宝档案

【场景3】爸爸登录后点击取消，展示妈妈的记录
───────────────────────────────────────────────────────────
✓ 爸爸登录成功 (dad@example.com)
✓ 登录返回: baby_id=1
  ├─ 预期: 点击取消后展示宝宝A的所有记录
  └─ 说明: baby_id已存在，无需创建新档案
✓ 时光轴数据获取成功
  ├─ 记录数量: 1条
  └─ [1] 2026-02-06 12:00:00 - 妈妈记录 - 妈妈

【场景4】爸爸新增一条宝宝A的记录
───────────────────────────────────────────────────────────
✓ 爸爸新增记录成功
  ├─ 内容: 爸爸记录 - 2026-02-06
  └─ 标签: 爸爸记录
✓ 时光轴记录更新成功 (共2条)
  ├─ [1] 2026-02-06 12:00:00 - 妈妈记录 - 妈妈
  └─ [2] 2026-02-06 12:01:00 - 爸爸记录 - 爸爸

【场景5】爸爸创建宝宝B的档案
───────────────────────────────────────────────────────────
✓ 宝宝B档案创建成功
  ├─ 用户: dad@example.com (爸爸)
  ├─ 宝宝ID: 2
  └─ 出生证编号: BABY002
✓ 爸爸可以看到宝宝列表 (共2个宝宝)
  ├─ [1] baby_id=1, name=李妈妈
  └─ [2] baby_id=2, name=宝宝B

【场景6】爸爸新增一条宝宝B的记录
───────────────────────────────────────────────────────────
✓ 爸爸新增宝宝B记录成功
  ├─ 内容: 爸爸新增宝宝B的记录 - 2026-02-06
  └─ 标签: 宝宝B

【场景7】妈妈登录后查看宝宝的记录
───────────────────────────────────────────────────────────
✓ 妈妈登录成功 (mom@example.com)
✓ 登录返回: baby_id=1
  ├─ 说明: 妈妈登录时返回的baby_id可能不同
  └─ 预期行为: 可以看到关联的宝宝列表
✓ 时光轴数据获取成功 (共2条记录)
  ├─ [1] 2026-02-06 12:00:00 - 妈妈记录 - 妈妈
  └─ [2] 2026-02-06 12:01:00 - 爸爸记录 - 爸爸

╔══════════════════════════════════════════════════════════╗
║                    所有测试场景完成                      ║
╚══════════════════════════════════════════════════════════╝
```

## 修复的问题

### 问题1: 奶奶注册后登录时仍显示弹窗

**原因**: 登录接口没有返回 `baby_id`

**修复**:
- 在 `Login` 函数中查询用户的宝宝列表
- 返回第一个宝宝的 ID

**文件**: `ad-backend/internal/handlers/auth.go`

### 问题2: TimelinePage 点击取消不生效

**原因**: 点击取消时没有正确处理已有关联宝宝的情况

**修复**:
- 检查 localStorage 中是否已存在 `baby_id`
- 如果存在，设置 `babyProfileSkipFetch` 标记
- TimelinePage 检测到该标记后直接从 localStorage 读取并展示记录

**文件**:
- `ad-frontend/src/pages/BabyProfilePage.tsx`
- `ad-frontend/src/pages/TimelinePage.tsx`

## 关键代码修改

### 后端 - auth.go

```go
// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
    // ... 查找用户 ...

    // 查询用户的宝宝列表（包括家庭成员的宝宝）
    var babyIDs []uint
    // ... 查询 family_members 表 ...
    // ... 查询 babies 表 ...

    // 取第一个宝宝ID（如果有）
    var babyID *uint
    if len(babyIDs) > 0 {
        babyID = &babyIDs[0]
    }

    response.Success(c, gin.H{
        "token":    token,
        "user_id":  user.ID,
        "email":    user.Email,
        "role":     user.Role,
        "baby_id":  babyID,  // 新增返回字段
    })
}
```

### 前端 - BabyProfilePage.tsx

```typescript
const navigateToTimeline = () => {
    const existingBabyId = localStorage.getItem('baby_id');

    if (existingBabyId) {
        // 已经有关联的宝宝，设置标记让 TimelinePage 知道跳过获取
        sessionStorage.setItem('babyProfileSkipFetch', 'true');
        if (onCancel) onCancel();
        navigate('/timeline', { replace: true });
    } else {
        // 没有关联的宝宝，设置标记让 TimelinePage 显示创建弹窗
        sessionStorage.setItem('babyProfileCreated', 'true');
        if (onCancel) onCancel();
        navigate('/timeline', { replace: true });
    }
};
```

### 前端 - TimelinePage.tsx

```typescript
useEffect(() => {
    // 如果有标记跳过获取（用户已关联宝宝点击取消）
    if (sessionStorage.getItem('babyProfileSkipFetch') === 'true') {
        sessionStorage.removeItem('babyProfileSkipFetch');
        const storedBabyId = localStorage.getItem('baby_id');
        if (storedBabyId) {
            const babyIdNum = parseInt(storedBabyId, 10);
            setBabyId(babyIdNum);
            fetchTimeline();
            fetchFamilyMembers();
        }
        return;
    }
    // ... 其他逻辑 ...
}, []);
```

## 注意事项

1. 测试数据库 `anan_diary_test` 需要手动创建
2. 测试过程中会保留测试数据，需要手动清理
3. 测试服务器使用端口 8081，与主服务器端口 8080 不冲突
4. 测试完成后，可以手动清理测试数据或重启测试服务器
