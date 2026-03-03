-- 给 users 表添加 name 列
-- 用户姓名，用于在界面上显示

ALTER TABLE users
ADD COLUMN IF NOT EXISTS name VARCHAR(100);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
