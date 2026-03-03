-- 清空数据库中所有数据
-- 删除所有表中的数据

-- 先删除外键约束，避免冲突
ALTER TABLE growth_records DROP CONSTRAINT IF EXISTS growth_records_baby_id_fkey CASCADE;
ALTER TABLE growth_records DROP CONSTRAINT IF EXISTS growth_records_user_id_fkey CASCADE;
ALTER TABLE babies DROP CONSTRAINT IF EXISTS babies_user_id_fkey CASCADE;
ALTER TABLE family_members DROP CONSTRAINT IF EXISTS family_members_baby_id_fkey CASCADE;
ALTER TABLE family_members DROP CONSTRAINT IF EXISTS fk_family_members_user_id CASCADE;

-- 删除所有表的数据（按依赖关系顺序）
TRUNCATE TABLE growth_records CASCADE;
TRUNCATE TABLE milestone_tags CASCADE;
TRUNCATE TABLE babies CASCADE;
TRUNCATE TABLE family_members CASCADE;
TRUNCATE TABLE users CASCADE;

-- 重新添加外键约束
ALTER TABLE growth_records ADD CONSTRAINT growth_records_baby_id_fkey
FOREIGN KEY (baby_id) REFERENCES babies(id) ON DELETE CASCADE;
ALTER TABLE growth_records ADD CONSTRAINT growth_records_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE babies ADD CONSTRAINT babies_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE family_members ADD CONSTRAINT family_members_baby_id_fkey
FOREIGN KEY (baby_id) REFERENCES babies(id) ON DELETE CASCADE;
ALTER TABLE family_members ADD CONSTRAINT fk_family_members_user_id
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 清理里程碑标签（因为TRUNCATE不会重置序列）
TRUNCATE TABLE milestone_tags RESTART IDENTITY CASCADE;

-- 完成提示
SELECT 'All data cleared from database' as status;
