-- 给 family_members 表添加 user_id 列
-- 用于关联注册用户，实现多对多关系

ALTER TABLE family_members
ADD COLUMN IF NOT EXISTS user_id BIGINT;

-- 添加外键约束
ALTER TABLE family_members
ADD CONSTRAINT fk_family_members_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 添加唯一约束：每个宝宝每种角色只能有一个
ALTER TABLE family_members
ADD CONSTRAINT uk_family_members_baby_role
UNIQUE (baby_id, role);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_family_members_user_id ON family_members(user_id);
