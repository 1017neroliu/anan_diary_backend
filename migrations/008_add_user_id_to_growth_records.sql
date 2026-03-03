-- 给 growth_records 表添加 user_id 列
-- 用于关联创建记录的用户，实现记录归属追踪

ALTER TABLE growth_records
ADD COLUMN IF NOT EXISTS user_id BIGINT;

-- 添加外键约束
ALTER TABLE growth_records
ADD CONSTRAINT fk_growth_records_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_growth_records_user_id ON growth_records(user_id);
