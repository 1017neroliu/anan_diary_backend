-- 1. 为 family_members 表添加 user_id 字段，用于关联注册用户
-- 2. 为 family_members 表添加角色唯一约束（一个宝宝只能有一个爸爸/妈妈/爷爷等）
--    但姐姐/哥哥可以有多个

-- 添加 user_id 字段
ALTER TABLE family_members ADD COLUMN IF NOT EXISTS user_id BIGINT;

-- 添加 user_id 外键约束
ALTER TABLE family_members
ADD CONSTRAINT fk_family_members_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 添加唯一约束：一个宝宝的同一个角色只能有一个
ALTER TABLE family_members
ADD CONSTRAINT uk_family_members_baby_role
UNIQUE (baby_id, role);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_family_members_user_id ON family_members(user_id);

-- 完成提示
SELECT 'Added user_id to family_members and created role uniqueness constraint' as status;
