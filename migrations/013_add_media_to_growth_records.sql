-- 添加媒体文件字段到成长记录表
ALTER TABLE growth_records ADD COLUMN IF NOT EXISTS media TEXT DEFAULT '[]';

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_growth_records_media ON growth_records(media);
