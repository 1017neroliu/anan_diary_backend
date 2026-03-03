-- 添加宝宝头像字段
ALTER TABLE babies ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(500) DEFAULT NULL;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_babies_avatar_url ON babies(avatar_url);
