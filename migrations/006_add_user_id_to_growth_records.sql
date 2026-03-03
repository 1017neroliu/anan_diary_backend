-- 为 growth_records 表添加 user_id 字段
ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS user_id BIGINT;
ALTER TABLE growth_records ADD CONSTRAINT fk_growth_records_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_growth_records_user_id ON growth_records(user_id);

-- 更新现有数据的 user_id（从 user_id 更新为对应的 user_id）
UPDATE growth_records
SET user_id = u.id
FROM babies b
JOIN users u ON b.user_id = u.id
WHERE growth_records.baby_id = b.id
AND growth_records.user_id IS NULL;

-- 完成提示
SELECT 'Added user_id to growth_records table' as status;
