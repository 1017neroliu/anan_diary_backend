-- 为 users 表添加 role 列
-- 用于存储家庭成员角色信息（爸爸/妈妈/爷爷/奶奶/姥姥/姥爷/哥哥/姐姐）

-- 添加 role 列
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT '爸爸';

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 更新已有记录的角色为默认值
UPDATE users SET role = '爸爸' WHERE role IS NULL OR role = '';

-- 完成提示
SELECT 'Added role column to users table' as status;
