-- 修改 birth_date 字段类型为 TIMESTAMP

-- 使用 ALTER COLUMN 修改类型（会自动转换数据）
ALTER TABLE babies
ALTER COLUMN birth_date TYPE TIMESTAMP
USING birth_date::TIMESTAMP;

-- 修改字段约束（NOT NULL 在类型转换后会自动调整）
ALTER TABLE babies
ALTER COLUMN birth_date SET NOT NULL;

-- 完成提示
SELECT 'Fixed birth_date field type to TIMESTAMP' as status;
